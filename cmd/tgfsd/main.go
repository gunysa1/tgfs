package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/gunysa1/tgfs/internal/bot"
	"github.com/gunysa1/tgfs/internal/cache"
	"github.com/gunysa1/tgfs/internal/config"
	"github.com/gunysa1/tgfs/internal/db"
	tgfsfs "github.com/gunysa1/tgfs/internal/fs"
	"github.com/gunysa1/tgfs/internal/ipc"
	"github.com/gunysa1/tgfs/internal/migrate"
)

func main() {
	cfgPath := flag.String("config", "/etc/tgfs/config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(cfg.DB.Path), 0755); err != nil {
		log.Fatalf("create db dir: %v", err)
	}
	if err := os.MkdirAll("/run/tgfs", 0755); err != nil {
		log.Fatalf("create run dir: %v", err)
	}

	database, err := db.Open(cfg.DB.Path)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer database.Close()

	botClient, err := bot.New(cfg.Telegram.BotToken)
	if err != nil {
		log.Fatalf("init bot: %v", err)
	}

	chunkCache := cache.New(int64(cfg.Cache.MaxSizeGB) * 1024 * 1024 * 1024)
	tgFS := tgfsfs.New(database, botClient, chunkCache)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handler := buildHandler(database, botClient, chunkCache, cfg)
	srv := ipc.NewServer("/run/tgfs/tgfs.sock", func(req ipc.Request) ipc.Response {
		if req.Command == "stop" {
			cancel()
			return ipc.Response{OK: true}
		}
		return handler(req)
	})
	go srv.Serve(ctx)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sig)
	go func() {
		select {
		case <-ctx.Done():
		case <-sig:
			cancel()
		}
	}()

	log.Printf("mounting tgfs at %s", cfg.Mount.Path)
	if err := tgfsfs.Mount(ctx, cfg.Mount.Path, tgFS); err != nil {
		log.Fatalf("mount: %v", err)
	}
}

func buildHandler(database *db.DB, botClient *bot.Client, chunkCache *cache.Cache, cfg *config.Config) ipc.HandlerFunc {
	return func(req ipc.Request) ipc.Response {
		switch req.Command {
		case "status":
			cur, max := chunkCache.Stats()
			return ipc.Response{OK: true, Data: map[string]string{
				"cache_used_mb": fmt.Sprintf("%d", cur/1024/1024),
				"cache_max_mb":  fmt.Sprintf("%d", max/1024/1024),
				"mount":         cfg.Mount.Path,
			}}

		case "cache.status":
			cur, max := chunkCache.Stats()
			return ipc.Response{OK: true, Data: map[string]string{
				"used_bytes": fmt.Sprintf("%d", cur),
				"max_bytes":  fmt.Sprintf("%d", max),
			}}

		case "cache.clear":
			chunkCache.Clear()
			return ipc.Response{OK: true}

		case "ls":
			path := req.Args["path"]
			children, err := database.ListChildren(path)
			if err != nil {
				return ipc.Response{Error: err.Error()}
			}
			var lines []string
			for _, f := range children {
				prefix := "-"
				if f.IsDir {
					prefix = "d"
				}
				lines = append(lines, fmt.Sprintf("%s %s (%d bytes)", prefix, f.Name, f.Size))
			}
			return ipc.Response{OK: true, Lines: lines}

		case "delete":
			path := req.Args["path"]
			f, err := database.GetFileByPath(path)
			if err != nil {
				return ipc.Response{Error: err.Error()}
			}
			chunks, err := database.GetChunksByFileID(f.ID)
			if err != nil {
				return ipc.Response{Error: err.Error()}
			}
			channels, err := database.ListChannels()
			if err != nil {
				return ipc.Response{Error: err.Error()}
			}
			chanMap := make(map[int64]int64) // channelID -> telegramID
			for _, ch := range channels {
				chanMap[ch.ID] = ch.TelegramID
			}
			for _, c := range chunks {
				if tgID, ok := chanMap[c.ChannelID]; ok {
					if err := botClient.DeleteMessage(tgID, c.MessageID); err != nil {
						log.Printf("delete message %d: %v", c.MessageID, err)
					}
				}
			}
			if err := database.DeleteFile(f.ID); err != nil {
				return ipc.Response{Error: err.Error()}
			}
			return ipc.Response{OK: true}

		case "mv":
			src, dst := req.Args["src"], req.Args["dst"]
			f, err := database.GetFileByPath(src)
			if err != nil {
				return ipc.Response{Error: err.Error()}
			}
			newName := filepath.Base(dst)
			if err := database.UpdateFilePath(f.ID, dst, newName); err != nil {
				return ipc.Response{Error: err.Error()}
			}
			return ipc.Response{OK: true}

		case "channel.add":
			tidStr := req.Args["telegram_id"]
			tid, err := strconv.ParseInt(tidStr, 10, 64)
			if err != nil {
				return ipc.Response{Error: "invalid telegram_id: " + tidStr}
			}
			if _, err := database.CreateChannel(db.Channel{TelegramID: tid, Name: req.Args["name"]}); err != nil {
				return ipc.Response{Error: err.Error()}
			}
			return ipc.Response{OK: true}

		case "channel.ls":
			channels, err := database.ListChannels()
			if err != nil {
				return ipc.Response{Error: err.Error()}
			}
			var lines []string
			for _, ch := range channels {
				lines = append(lines, fmt.Sprintf("%s (telegram_id: %d)", ch.Name, ch.TelegramID))
			}
			return ipc.Response{OK: true, Lines: lines}

		case "channel.rm":
			ch, err := database.GetChannelByName(req.Args["name"])
			if err != nil {
				return ipc.Response{Error: "channel not found: " + req.Args["name"]}
			}
			if err := database.DeleteChannel(ch.ID); err != nil {
				return ipc.Response{Error: err.Error()}
			}
			return ipc.Response{OK: true}

		case "upload":
			localPath := req.Args["local"]
			virtualPath := req.Args["virtual"]
			name := req.Args["name"]

			channels, err := database.ListChannels()
			if err != nil || len(channels) == 0 {
				return ipc.Response{Error: "no channels configured"}
			}
			ch := channels[0]
			chunkSizeBytes := int64(cfg.Chunk.SizeMB) * 1024 * 1024

			info, err := os.Stat(localPath)
			if err != nil {
				return ipc.Response{Error: fmt.Sprintf("stat %q: %v", localPath, err)}
			}

			dbFile, err := database.CreateFile(db.File{
				Path:     virtualPath,
				Name:     name,
				Size:     info.Size(),
				MimeType: "application/octet-stream",
			})
			if err != nil {
				return ipc.Response{Error: err.Error()}
			}

			chunks := migrate.PlanChunks(info.Size(), chunkSizeBytes)
			go func() {
				for _, chunk := range chunks {
					sr, f, err := migrate.OpenChunk(localPath, chunk.Offset, chunk.Size)
					if err != nil {
						log.Printf("upload: open chunk: %v", err)
						return
					}
					fname := migrate.ChunkFilename(virtualPath, chunk.Index, len(chunks))
					msgID, tgFileID, err := botClient.Upload(ch.TelegramID, fname, sr)
					f.Close()
					if err != nil {
						log.Printf("upload: telegram: %v", err)
						return
					}
					if _, err := database.CreateChunk(db.Chunk{
						FileID:         dbFile.ID,
						ChunkIndex:     chunk.Index,
						MessageID:      msgID,
						TelegramFileID: tgFileID,
						ChannelID:      ch.ID,
						Size:           chunk.Size,
					}); err != nil {
						log.Printf("upload: save chunk: %v", err)
						return
					}
				}
				log.Printf("uploaded %s", virtualPath)
			}()
			return ipc.Response{OK: true}

		case "migrate":
			localPath := req.Args["path"]
			dryRun := req.Args["dry_run"] == "true"
			chunkSizeBytes := int64(cfg.Chunk.SizeMB) * 1024 * 1024

			channels, err := database.ListChannels()
			if err != nil || len(channels) == 0 {
				return ipc.Response{Error: "no channels configured"}
			}
			defaultChannel := channels[0]

			go func() {
				m := &migrate.Migrator{
					ChunkSizeBytes: chunkSizeBytes,
					DryRun:         dryRun,
					OnProgress: func(vpath string, done, total int) {
						log.Printf("[%d/%d] %s", done, total, vpath)
					},
					OnSkip: func(vpath string) {
						log.Printf("skip (exists): %s", vpath)
					},
				}
				err := m.Run(
					localPath,
					func(vpath string) bool {
						_, err := database.GetFileByPath(vpath)
						return err == nil
					},
					func(vpath, name string) error {
						_, err := database.CreateFile(db.File{Path: vpath, Name: name, IsDir: true})
						return err
					},
					func(entry migrate.Entry) error {
						chunks := migrate.PlanChunks(entry.Size, chunkSizeBytes)
						dbFile, err := database.CreateFile(db.File{
							Path:     entry.VirtualPath,
							Name:     entry.Name,
							Size:     entry.Size,
							MimeType: entry.MimeType,
						})
						if err != nil {
							return err
						}
						for _, chunk := range chunks {
							sr, f, err := migrate.OpenChunk(entry.LocalPath, chunk.Offset, chunk.Size)
							if err != nil {
								return err
							}
							fname := migrate.ChunkFilename(entry.VirtualPath, chunk.Index, len(chunks))
							msgID, tgFileID, err := botClient.Upload(defaultChannel.TelegramID, fname, sr)
							f.Close()
							if err != nil {
								return err
							}
							if _, err = database.CreateChunk(db.Chunk{
								FileID:         dbFile.ID,
								ChunkIndex:     chunk.Index,
								MessageID:      msgID,
								TelegramFileID: tgFileID,
								ChannelID:      defaultChannel.ID,
								Size:           chunk.Size,
							}); err != nil {
								return err
							}
						}
						return nil
					},
				)
				if err != nil {
					log.Printf("migrate error: %v", err)
				} else {
					log.Printf("migrate complete")
				}
			}()
			return ipc.Response{OK: true}

		default:
			return ipc.Response{Error: fmt.Sprintf("unknown command: %s", req.Command)}
		}
	}
}
