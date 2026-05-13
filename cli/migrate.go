package cli

import (
	"fmt"

	"github.com/gunysa1/tgfs/internal/ipc"
	"github.com/spf13/cobra"
)

func newMigrateCmd() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "migrate <local-path>",
		Short: "Bulk upload an existing library to Telegram",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ipc.NewClient(flagSocket)
			if err != nil {
				return err
			}
			dryRunStr := "false"
			if dryRun {
				dryRunStr = "true"
			}
			resp, err := client.Send(ipc.Request{Command: "migrate", Args: map[string]string{
				"path":    args[0],
				"dry_run": dryRunStr,
			}})
			if err != nil {
				return err
			}
			if !resp.OK {
				return fmt.Errorf("migrate failed: %s", resp.Error)
			}
			fmt.Println("migration started — check daemon logs for progress")
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be uploaded without uploading")
	return cmd
}
