package cli

import (
	"fmt"
	"path/filepath"

	"github.com/gunysa1/tgfs/internal/ipc"
	"github.com/spf13/cobra"
)

func newFileCmds() []*cobra.Command {
	ls := &cobra.Command{
		Use:   "ls [path]",
		Short: "List files in the virtual filesystem",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "/"
			if len(args) > 0 {
				path = args[0]
			}
			client, err := ipc.NewClient(flagSocket)
			if err != nil {
				return err
			}
			resp, err := client.Send(ipc.Request{Command: "ls", Args: map[string]string{"path": path}})
			if err != nil {
				return err
			}
			for _, line := range resp.Lines {
				fmt.Println(line)
			}
			return nil
		},
	}

	del := &cobra.Command{
		Use:   "delete <virtual-path>",
		Short: "Delete a file from the virtual filesystem and Telegram",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ipc.NewClient(flagSocket)
			if err != nil {
				return err
			}
			resp, err := client.Send(ipc.Request{Command: "delete", Args: map[string]string{"path": args[0]}})
			if err != nil {
				return err
			}
			if !resp.OK {
				return fmt.Errorf("delete failed: %s", resp.Error)
			}
			fmt.Printf("deleted %s\n", args[0])
			return nil
		},
	}

	mv := &cobra.Command{
		Use:   "mv <src> <dst>",
		Short: "Rename/move a file in the virtual filesystem (no re-upload)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ipc.NewClient(flagSocket)
			if err != nil {
				return err
			}
			resp, err := client.Send(ipc.Request{Command: "mv", Args: map[string]string{"src": args[0], "dst": args[1]}})
			if err != nil {
				return err
			}
			if !resp.OK {
				return fmt.Errorf("mv failed: %s", resp.Error)
			}
			fmt.Printf("moved %s → %s\n", args[0], args[1])
			return nil
		},
	}

	upload := &cobra.Command{
		Use:   "upload <local-file> <virtual-path>",
		Short: "Upload a local file to the virtual filesystem",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ipc.NewClient(flagSocket)
			if err != nil {
				return err
			}
			resp, err := client.Send(ipc.Request{Command: "upload", Args: map[string]string{
				"local":   args[0],
				"virtual": args[1],
				"name":    filepath.Base(args[0]),
			}})
			if err != nil {
				return err
			}
			if !resp.OK {
				return fmt.Errorf("upload failed: %s", resp.Error)
			}
			fmt.Printf("uploaded %s → %s\n", args[0], args[1])
			return nil
		},
	}

	return []*cobra.Command{ls, del, mv, upload}
}
