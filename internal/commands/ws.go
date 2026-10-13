package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// WsCmd outputs the workspace as JSON data
var WsCmd = &cobra.Command{
	Use:   "ws",
	Short: "Show the workspace as data",
	Long: `Output the workspace state as JSON.

Use --get to query specific keys in the workspace structure.
Use --out to write the output to a file.

Examples:
  goliath ws                      Output entire workspace as JSON
  goliath ws --get components     Output only the components
  goliath ws --get settings       Output only the settings
  goliath ws --out ws.json        Write workspace to file`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := loadWorkspaceFromArgs(cmd)
		if err != nil {
			return err
		}

		get, _ := cmd.Flags().GetString("get")
		out, _ := cmd.Flags().GetString("out")

		var data []byte
		if get != "" {
			data, err = getWorkspaceKey(ws, get)
		} else {
			data, err = ws.ToJSON()
		}
		if err != nil {
			return err
		}

		if out != "" {
			if err := os.WriteFile(out, data, 0644); err != nil {
				return fmt.Errorf("failed to write output file: %w", err)
			}
			fmt.Printf("Workspace written to %s\n", out)
		} else {
			fmt.Println(string(data))
		}

		return nil
	},
}

func init() {
	WsCmd.Flags().String("ws-dir", "", "Path to workspace directory")
	WsCmd.Flags().String("get", "", "Get a specific key from the workspace (e.g., components, settings)")
	WsCmd.Flags().String("out", "", "Write output to file")
}

// getWorkspaceKey extracts a specific key path from the workspace JSON
func getWorkspaceKey(ws interface{}, keyPath string) ([]byte, error) {
	// First, marshal the whole workspace
	fullData, err := json.Marshal(ws)
	if err != nil {
		return nil, err
	}

	// Unmarshal into a generic map
	var data map[string]interface{}
	if err := json.Unmarshal(fullData, &data); err != nil {
		return nil, err
	}

	// Navigate the key path (colon-separated)
	keys := strings.Split(keyPath, ":")
	current := interface{}(data)

	for _, key := range keys {
		switch v := current.(type) {
		case map[string]interface{}:
			if key == "keys" {
				// Special: return list of keys
				var keysList []string
				for k := range v {
					keysList = append(keysList, k)
				}
				return json.MarshalIndent(keysList, "", "  ")
			}
			if key == "count" {
				return json.MarshalIndent(len(v), "", "  ")
			}
			val, ok := v[key]
			if !ok {
				return nil, fmt.Errorf("key %q not found", key)
			}
			current = val
		default:
			return nil, fmt.Errorf("cannot navigate into non-object at key %q", key)
		}
	}

	return json.MarshalIndent(current, "", "  ")
}
