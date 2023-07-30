/*
Copyright © 2023 Nick Wright <nwright970@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"log"

	"github.com/redgoat650/barnacle-net/internal/client"
	"github.com/spf13/cobra"
)

const (
	refreshFlagKey   = "refresh"
	refreshFlagAlias = "r"
)

// barnacleListCmd represents the list command
var barnacleListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "Query a list of barnacles connected to the net.",
	Long:    `Query a list of barnacles connected to the net.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Println("list called")

		r, err := cmd.Flags().GetBool(refreshFlagKey)
		if err != nil {
			return err
		}

		err = client.ListNodes(r)
		if err != nil {
			log.Println("list nodes returned error:", err)
		}

		return nil
	},
}

func init() {
	barnacleCmd.AddCommand(barnacleListCmd)

	barnacleListCmd.Flags().BoolP(refreshFlagKey, refreshFlagAlias, false, "Re-identify all connected nodes. If false, just returns current server state.")
}
