package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(payeeLocationCmd)
	payeeLocationCmd.AddCommand(payeeLocationListCmd)
	payeeLocationCmd.AddCommand(payeeLocationGetCmd)
}

var payeeLocationCmd = &cobra.Command{
	Use:   "payee-location",
	Short: "View payee locations",
	Long: `List and view payee GPS locations recorded by YNAB mobile apps.

Examples:
  nab payee-location list                    List all payee locations
  nab payee-location list --payee <payee-id> List locations for a specific payee
  nab payee-location get <location-id>       Get a specific payee location`,
}

var payeeLocationListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all payee locations",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newAPIClient()
		if err != nil {
			return err
		}

		budgetID, err := requireBudget()
		if err != nil {
			return err
		}

		payeeID, _ := cmd.Flags().GetString("payee")

		var path string
		if payeeID != "" {
			path = fmt.Sprintf("/budgets/%s/payees/%s/payee_locations", budgetID, payeeID)
		} else {
			path = fmt.Sprintf("/budgets/%s/payee_locations", budgetID)
		}

		var result struct {
			PayeeLocations []map[string]any `json:"payee_locations"`
		}
		if err := client.GetJSON(path, &result); err != nil {
			return fmt.Errorf("listing payee locations: %w", err)
		}

		return printAPIResult(toAnySlice(result.PayeeLocations))
	},
}

var payeeLocationGetCmd = &cobra.Command{
	Use:   "get <payee-location-id>",
	Short: "Get payee location details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newAPIClient()
		if err != nil {
			return err
		}

		budgetID, err := requireBudget()
		if err != nil {
			return err
		}

		var result struct {
			PayeeLocation map[string]any `json:"payee_location"`
		}
		if err := client.GetJSON(fmt.Sprintf("/budgets/%s/payee_locations/%s", budgetID, args[0]), &result); err != nil {
			return fmt.Errorf("getting payee location: %w", err)
		}

		return printAPIResult(result.PayeeLocation)
	},
}

func init() {
	payeeLocationListCmd.Flags().String("payee", "", "Filter by payee ID")
}
