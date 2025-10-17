package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/LoriKarikari/compak/internal/core/index"
)

var searchCmd = &cobra.Command{
	Use:   "search [QUERY]",
	Short: "Search for compak paks",
	Long: `Search for compak paks in the pak index.

Examples:
  compak search nginx
  compak search postgres --limit 20
  compak search blog
  compak search database`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := ""
		if len(args) > 0 {
			query = args[0]
		}

		limit, err := cmd.Flags().GetInt("limit")
		if err != nil {
			return fmt.Errorf("failed to get limit flag: %w", err)
		}

		return searchPackages(query, limit)
	},
}

func init() {
	searchCmd.Flags().Int("limit", 10, "Maximum number of results to show")
	rootCmd.AddCommand(searchCmd)
}

func searchPackages(query string, limit int) error {
	if query != "" {
		fmt.Printf("Searching for paks matching '%s'...\n\n", query)
	} else {
		fmt.Printf("Listing available paks...\n\n")
	}

	client := index.NewClient()
	ctx := context.Background()

	results, err := client.Search(ctx, query, limit)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(results) == 0 {
		if query != "" {
			fmt.Printf("No paks found matching '%s'.\n", query)
		} else {
			fmt.Println("No paks available in the index.")
		}
		fmt.Println("\nTo add a pak to the index, see:")
		fmt.Println("  https://github.com/LoriKarikari/compak")
		return nil
	}

	fmt.Printf("Found %d pak(s):\n\n", len(results))

	for _, result := range results {
		displaySearchResult(result)
	}

	fmt.Printf("\nTo install a pak, run:\n")
	fmt.Printf("  compak install PAK_NAME\n")

	return nil
}

func displaySearchResult(result index.SearchResult) {
	version := result.Version
	if version != "latest" && version[0] != 'v' {
		version = "v" + version
	}
	fmt.Printf("[%s] %s\n", result.Name, version)

	if result.Description != "" {
		fmt.Printf("   %s\n", result.Description)
	}

	if result.Author != "" {
		fmt.Printf("   Author: %s\n", result.Author)
	}

	if result.Homepage != "" {
		fmt.Printf("   Homepage: %s\n", result.Homepage)
	}

	fmt.Println()
}
