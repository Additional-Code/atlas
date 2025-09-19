package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/fx"

	"github.com/Additional-Code/atlas/internal/app"
	"github.com/Additional-Code/atlas/internal/migration"
	"github.com/Additional-Code/atlas/internal/seeder"
)

// NewRootCommand builds the root atlas CLI command.
func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "atlas",
		Short: "Atlas developer toolkit",
	}

	root.AddCommand(newStartCmd())
	root.AddCommand(newMigrateCmd())
	root.AddCommand(newSeedCmd())
	root.AddCommand(newModuleCmd())
	root.AddCommand(newWorkerCmd())

	return root
}

// Execute runs the atlas CLI.
func Execute() error {
	if err := NewRootCommand().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return err
	}
	return nil
}

func newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "start",
		Aliases: []string{"run"},
		Short:   "Run the HTTP service",
		RunE: func(cmd *cobra.Command, args []string) error {
			application := fx.New(app.Module)
			if err := application.Start(cmd.Context()); err != nil {
				return err
			}
			<-cmd.Context().Done()
			stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			return application.Stop(stopCtx)
		},
	}
}

func newMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Run database migrations",
	}

	upCmd := &cobra.Command{
		Use:   "up",
		Short: "Apply migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			var mig *migration.Migrator
			opts := fx.Options(app.Core, migration.Module, fx.Populate(&mig))
			return runWithApp(cmd.Context(), opts, func(ctx context.Context) error {
				if err := mig.Up(ctx); err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), "migrations applied")
				return nil
			})
		},
	}

	downCmd := &cobra.Command{
		Use:   "down",
		Short: "Rollback migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			steps, _ := cmd.Flags().GetInt("steps")
			all, _ := cmd.Flags().GetBool("all")
			var mig *migration.Migrator
			opts := fx.Options(app.Core, migration.Module, fx.Populate(&mig))
			return runWithApp(cmd.Context(), opts, func(ctx context.Context) error {
				if err := mig.Down(ctx, steps, all); err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), "migrations rolled back")
				return nil
			})
		},
	}
	downCmd.Flags().Int("steps", 1, "Number of migration steps to rollback")
	downCmd.Flags().Bool("all", false, "Rollback all applied migrations")

	cmd.AddCommand(upCmd, downCmd)
	return cmd
}

func newSeedCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "seed",
		Short: "Run database seeders",
		RunE: func(cmd *cobra.Command, args []string) error {
			var seed *seeder.Seeder
			opts := fx.Options(app.Core, seeder.Module, fx.Populate(&seed))
			return runWithApp(cmd.Context(), opts, func(ctx context.Context) error {
				if err := seed.Orders(ctx); err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), "seed data applied")
				return nil
			})
		},
	}
}

func newModuleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "module",
		Short: "Scaffold modules",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "create [name]",
		Short: "Create a new domain module",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			fmt.Fprintf(cmd.OutOrStdout(), "module %s scaffolded (placeholder)\n", name)
			return nil
		},
	})
	return cmd
}

func newWorkerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "worker",
		Short: "Manage background workers",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "run",
		Short: "Run worker engine",
		RunE: func(cmd *cobra.Command, args []string) error {
			application := fx.New(app.Worker)
			if err := application.Start(cmd.Context()); err != nil {
				return err
			}
			<-cmd.Context().Done()
			stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			return application.Stop(stopCtx)
		},
	})
	return cmd
}

func runWithApp(ctx context.Context, opts fx.Option, fn func(context.Context) error) error {
	application := fx.New(opts, fx.NopLogger)
	if err := application.Start(ctx); err != nil {
		return err
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = application.Stop(stopCtx)
	}()
	return fn(ctx)
}
