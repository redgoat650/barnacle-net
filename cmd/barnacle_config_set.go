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
	"fmt"
	"log"

	"github.com/redgoat650/barnacle-net/internal/client"
	"github.com/redgoat650/barnacle-net/internal/deploy"
	"github.com/spf13/cobra"
)

// barnacleConfigSetCmd represents the set command
var barnacleConfigSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set configuration for a node",
	Long:  `Set configuration for a node.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("set called", args)

		nodeDeploySettings, err := deploy.GetValidNodeDeploySettings()
		if err != nil {
			return fmt.Errorf("getting and validating node configs: %s", err)
		}

		if err := client.ConfigSet(nodeDeploySettings...); err != nil {
			log.Println("error setting config:", err)
		}

		return nil
	},
}

func init() {
	barnacleConfigCmd.AddCommand(barnacleConfigSetCmd)
}
