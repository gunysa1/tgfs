package cli

import (
	"fmt"

	"github.com/gunysa1/tgfs/internal/ipc"
	"github.com/spf13/cobra"
)

func newCacheCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage the chunk cache",
	}

	status := &cobra.Command{
		Use:   "status",
		Short: "Show cache usage",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ipc.NewClient(flagSocket)
			if err != nil {
				return err
			}
			resp, err := client.Send(ipc.Request{Command: "cache.status"})
			if err != nil {
				return err
			}
			for k, v := range resp.Data {
				fmt.Printf("%s: %s\n", k, v)
			}
			return nil
		},
	}

	clear := &cobra.Command{
		Use:   "clear",
		Short: "Evict all cached chunks",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ipc.NewClient(flagSocket)
			if err != nil {
				return err
			}
			_, err = client.Send(ipc.Request{Command: "cache.clear"})
			return err
		},
	}

	cmd.AddCommand(status, clear)
	return cmd
}
