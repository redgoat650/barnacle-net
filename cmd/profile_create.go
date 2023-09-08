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

// profileCreateCmd represents the create command
var profileCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a profile",
	Long:  `Create a profile.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("create called")

		name, err := cmd.Flags().GetString("name")
		if err != nil {
			return err
		}

		if name == "" {
			return errors.New("name must be provided")
		}

		chain, err := cmd.Flags().GetString("chain")
		if err != nil {
			return err
		}

		if chain == "" {
			return errors.New("chain must be specified")
		}

		key, err := cmd.Flags().GetString("apikey")
		if err != nil {
			return err
		}

		if err := client.CreateBCProfile(name, chain, key); err != nil {
			log.Printf("error creating loopring profile: %s", err)
		}

		return nil
	},
}

func init() {
	profileCmd.AddCommand(profileCreateCmd)

	profileCreateCmd.Flags().StringP("apikey", "k", "", "API key for accessing loopring API")
	profileCreateCmd.Flags().StringP("name", "n", "", "Name for profile")
	profileCreateCmd.Flags().StringP("chain", "c", "loopring", "Chain profile corresponds to")
}
