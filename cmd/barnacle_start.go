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

	"github.com/redgoat650/barnacle-net/internal/barnacle"
	"github.com/redgoat650/barnacle-net/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	nodeNameFlagName    = "name"
	nodeNameShorthand   = "n"
	nodeOrientFlagName  = "orientation"
	nodeOrientShorthand = "o"
	nodeLabelsFlagName  = "labels"
	nodeLabelsShorthand = "l"
)

// barnacleStartCmd represents the start command
var barnacleStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a barnacle node",
	Long:  `Start a barnacle node.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Println("start called")

		return barnacle.RunBarnacle()
	},
}

func init() {
	barnacleCmd.AddCommand(barnacleStartCmd)

	barnacleStartCmd.Flags().StringP(nodeNameFlagName, nodeNameShorthand, "", "node name")
	barnacleStartCmd.Flags().StringP(nodeOrientFlagName, nodeOrientShorthand, "", "node orientation based on button position [u,d,l,r]")
	barnacleStartCmd.Flags().StringSliceP(nodeLabelsFlagName, nodeLabelsShorthand, nil, "list of aliases for a node")

	viper.BindPFlag(config.NodeNameConfigKey, barnacleStartCmd.Flags().Lookup(nodeNameFlagName))
	viper.BindPFlag(config.NodeOrientationConfigKey, barnacleStartCmd.Flags().Lookup(nodeOrientFlagName))
	viper.BindPFlag(config.NodeLabelsConfigKey, barnacleStartCmd.Flags().Lookup(nodeLabelsFlagName))
}
