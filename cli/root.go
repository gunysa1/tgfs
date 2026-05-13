package cli

import "github.com/spf13/cobra"

var (
	flagConfig string
	flagSocket string
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "tgfs",
		Short: "Telegram-backed FUSE storage for Plex",
	}
	root.PersistentFlags().StringVar(&flagConfig, "config", "/etc/tgfs/config.yaml", "path to config file")
	root.PersistentFlags().StringVar(&flagSocket, "socket", "/run/tgfs/tgfs.sock", "path to daemon socket")

	root.AddCommand(newDaemonCmds()...)
	root.AddCommand(newFileCmds()...)
	root.AddCommand(newChannelCmd())
	root.AddCommand(newCacheCmd())
	root.AddCommand(newMigrateCmd())

	return root
}
