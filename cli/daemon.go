package cli

import (
	"fmt"

	"github.com/gunysa1/tgfs/internal/ipc"
	"github.com/spf13/cobra"
)

func newDaemonCmds() []*cobra.Command {
	stop := &cobra.Command{
		Use:   "stop",
		Short: "Stop the tgfs daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ipc.NewClient(flagSocket)
			if err != nil {
				return err
			}
			_, err = client.Send(ipc.Request{Command: "stop"})
			return err
		},
	}

	status := &cobra.Command{
		Use:   "status",
		Short: "Show daemon status",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ipc.NewClient(flagSocket)
			if err != nil {
				return fmt.Errorf("daemon not running: %w", err)
			}
			resp, err := client.Send(ipc.Request{Command: "status"})
			if err != nil {
				return err
			}
			for k, v := range resp.Data {
				fmt.Printf("%s: %s\n", k, v)
			}
			return nil
		},
	}

	return []*cobra.Command{stop, status}
}
