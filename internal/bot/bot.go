package bot

import (
	"fmt"
	"io"
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Client struct {
	api *tgbotapi.BotAPI
}

func New(token string) (*Client, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("init bot api: %w", err)
	}
	return &Client{api: api}, nil
}

// Upload sends a file chunk to the given Telegram channel.
// Returns (messageID, telegramFileID, error).
// telegramFileID is needed later for DownloadByFileID.
func (c *Client) Upload(channelID int64, filename string, r io.Reader, size int64) (int, string, error) {
	reader := tgbotapi.FileReader{
		Name:   filename,
		Reader: r,
	}
	msg := tgbotapi.NewDocument(channelID, reader)
	msg.Caption = filename
	sent, err := c.api.Send(msg)
	if err != nil {
		return 0, "", fmt.Errorf("upload chunk %q: %w", filename, err)
	}
	fileID := ""
	if sent.Document != nil {
		fileID = sent.Document.FileID
	}
	return sent.MessageID, fileID, nil
}

// DownloadByFileID downloads a file using its Telegram file_id.
// Caller must close the returned ReadCloser.
func (c *Client) DownloadByFileID(fileID string) (io.ReadCloser, error) {
	file, err := c.api.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return nil, fmt.Errorf("get file %q: %w", fileID, err)
	}
	url := file.Link(c.api.Token)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("download file: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("download file: HTTP %d", resp.StatusCode)
	}
	return resp.Body, nil
}

// DeleteMessage removes the Telegram message containing a chunk.
func (c *Client) DeleteMessage(channelID int64, messageID int) error {
	cfg := tgbotapi.NewDeleteMessage(channelID, messageID)
	_, err := c.api.Request(cfg)
	if err != nil {
		return fmt.Errorf("delete message %d: %w", messageID, err)
	}
	return nil
}
