package commands

import (
	"errors"
	"github.com/cortezaproject/corteza-server/compose/importer"
	"github.com/cortezaproject/corteza-server/compose/service"
	"github.com/cortezaproject/corteza-server/compose/types"
	"github.com/cortezaproject/corteza-server/pkg/auth"
	"github.com/cortezaproject/corteza-server/pkg/cli"
	"github.com/cortezaproject/corteza-server/store"
	"github.com/spf13/cobra"
	"io"
	"os"
	"strconv"
)

func Importer() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import",

		Run: func(cmd *cobra.Command, args []string) {
			var (
				ctx    = auth.SetSuperUserContext(cli.Context())
				ff     []io.Reader
				nsFlag = cmd.Flags().Lookup("namespace").Value.String()
				ns     *types.Namespace
				err    error
			)

			if nsFlag != "" {
				if namespaceID, _ := strconv.ParseUint(nsFlag, 10, 64); namespaceID > 0 {
					ns, err = service.DefaultNamespace.FindByID(namespaceID)
					if errors.Is(err, store.ErrNotFound) {
						cli.HandleError(err)
					}
				} else if ns, err = service.DefaultNamespace.FindByHandle(nsFlag); err != nil {
					if errors.Is(err, store.ErrNotFound) {
						cli.HandleError(err)
					}
				}
			}

			if len(args) > 0 {
				ff = make([]io.Reader, len(args))
				for a, arg := range args {
					ff[a], err = os.Open(arg)
					cli.HandleError(err)
				}
				cli.HandleError(importer.Import(ctx, ns, ff...))
			} else {
				cli.HandleError(importer.Import(ctx, ns, os.Stdin))
			}
		},
	}

	cmd.Flags().String("namespace", "", "Import into namespace (by ID or string)")

	return cmd
}
