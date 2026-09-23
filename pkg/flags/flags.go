// Package flags parses and validates command-line options for collage generation.
package flags

import (
	"fmt"
	"natasha-audrey/lastfm-collage-generator/pkg/config/timeframe"

	"github.com/spf13/cobra"
)

// Flags holds the parsed collage options.
type Flags struct {
	// Time selects the listening period.
	Time timeframe.TimeFrame
	// Size is the number of rows and columns, from 3 to 10.
	Size int
	// Path is the output image filename.
	Path string
	// User is the Last.fm username to query.
	User string
}

// NewCommand builds the CLI and calls run with validated collage options.
// Help and version requests skip validation and collage generation.
func NewCommand(version string, run func(*Flags) error) *cobra.Command {
	options := &Flags{}
	var period string
	cmd := &cobra.Command{
		Use:           "lastfm-collage-generator",
		Short:         "Generate a collage of your top Last.fm albums",
		Version:       version,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			options.Time, err = timeframe.ParseString(period)
			if err != nil {
				return err
			}
			if options.Size < 3 || options.Size > 10 {
				return fmt.Errorf("size %v needs to be between 3 and 10", options.Size)
			}
			options.Path, err = parsePath(options.Path)
			if err != nil {
				return err
			}
			if options.User == "" {
				options.User = "tashayasha"
			}
			cmd.SilenceUsage = true
			return run(options)
		},
	}
	cmd.SetVersionTemplate("{{.Version}}\n")
	cmd.Flags().StringVarP(&options.User, "user", "u", "tashayasha", "The user to query")
	cmd.Flags().StringVarP(&period, "timeframe", "t", "7day", "The listening period: 7day, 1month, 3month, 6month, 12month, overall")
	cmd.Flags().IntVarP(&options.Size, "size", "s", 5, "Sets the size x size of the collage (3-10)")
	cmd.Flags().StringVarP(&options.Path, "path", "p", "./collage.png", "The path the collage is written to")
	cmd.Flags().BoolP("version", "v", false, "Prints the CLI version")
	return cmd
}
