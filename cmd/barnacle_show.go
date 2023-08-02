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
	"errors"
	"fmt"
	"log"

	"github.com/redgoat650/barnacle-net/internal/client"
	"github.com/spf13/cobra"
)

// barnacleShowCmd represents the show command
var barnacleShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show the specified image on a barnacle node.",
	Long: `Show the specified image on a barnacle node.
A node may be optionally specified. If no node is specified,
the image will be displayed on a random node.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("show called")

		if len(args) < 1 {
			return errors.New("image argument required")
		}

		fmt.Println("Args", args)
		node, err := cmd.Flags().GetString("node")
		if err != nil {
			return err
		}

		err = client.ShowImage(node, args...)
		if err != nil {
			log.Println("show image returned error:", err)
		}

		return nil
	},
}

func init() {
	barnacleCmd.AddCommand(barnacleShowCmd)

	barnacleShowCmd.Flags().StringP("node", "n", "", "Identifier of the specific node to display on.")
}
