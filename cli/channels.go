package cli

import (
	"fmt"

	"github.com/gunysa1/tgfs/internal/ipc"
	"github.com/spf13/cobra"
)

func newChannelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "channel",
		Short: "Manage Telegram storage channels",
	}

	add := &cobra.Command{
		Use:   "add <telegram-channel-id> <name>",
		Short: "Register a Telegram channel for storage",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ipc.NewClient(flagSocket)
			if err != nil {
				return err
			}
			resp, err := client.Send(ipc.Request{Command: "channel.add", Args: map[string]string{
				"telegram_id": args[0],
				"name":        args[1],
			}})
			if err != nil {
				return err
			}
			if !resp.OK {
				return fmt.Errorf("add channel failed: %s", resp.Error)
			}
			fmt.Printf("registered channel %s (id: %s)\n", args[1], args[0])
			return nil
		},
	}

	list := &cobra.Command{
		Use:   "ls",
		Short: "List registered channels",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ipc.NewClient(flagSocket)
			if err != nil {
				return err
			}
			resp, err := client.Send(ipc.Request{Command: "channel.ls"})
			if err != nil {
				return err
			}
			for _, line := range resp.Lines {
				fmt.Println(line)
			}
			return nil
		},
	}

	rm := &cobra.Command{
		Use:   "rm <name>",
		Short: "Unregister a channel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ipc.NewClient(flagSocket)
			if err != nil {
				return err
			}
			resp, err := client.Send(ipc.Request{Command: "channel.rm", Args: map[string]string{"name": args[0]}})
			if err != nil {
				return err
			}
			if !resp.OK {
				return fmt.Errorf("rm channel failed: %s", resp.Error)
			}
			fmt.Printf("removed channel %s\n", args[0])
			return nil
		},
	}

	cmd.AddCommand(add, list, rm)
	return cmd
}
