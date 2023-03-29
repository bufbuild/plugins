// Copyright 2020-2023 Buf Technologies, Inc.
//
// All rights reserved.

package bufwasmtool

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufwasm"
	wasmpluginv1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/wasmplugin/v1"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Main is the main.
func Main(name string) {
	appcmd.Main(context.Background(), newRootCommand(name))
}

func newRootCommand(name string) *appcmd.Command {
	builder := appflag.NewBuilder(name)
	return &appcmd.Command{
		Use: name,
		SubCommands: []*appcmd.Command{
			newBufsectionCmd(builder),
			newExecuteCmd(builder),
			newInspectCmd(builder),
			newNameCmd(builder),
		},
		BindPersistentFlags: builder.BindRoot,
	}
}

type bufsectionFlags struct {
	Abi   wasmpluginv1.WasmABI
	Files map[string]string
}

func (f *bufsectionFlags) Bind(flagSet *pflag.FlagSet) {
	abiFlagVar := NewProtoEnumFlag[wasmpluginv1.WasmABI]()
	flagSet.VarP(
		abiFlagVar,
		"abi",
		"a",
		fmt.Sprintf("one of [%s]", strings.Join(abiFlagVar.Allowed, ", ")),
	)
	flagSet.StringToStringVar(
		&f.Files,
		"file",
		nil,
		"bundle a file into the wasm file with buf extensions. Syntax: /path/to/file=/path/in/wasm",
	)
}

func newBufsectionCmd(builder appflag.Builder) *appcmd.Command {
	flags := new(bufsectionFlags)
	return &appcmd.Command{
		Use:       "bufsection",
		Short:     "Create a bufsection to append to an existing wasm file",
		Args:      cobra.ArbitraryArgs,
		BindFlags: flags.Bind,
		Run: builder.NewRunFunc(func(ctx context.Context, c appflag.Container) error {
			metadata := &wasmpluginv1.ExecConfig{}
			if flags.Abi != wasmpluginv1.WasmABI_WASM_ABI_UNSPECIFIED {
				metadata.WasmAbi = flags.Abi
			}
			for i := 0; i < c.NumArgs(); i++ {
				metadata.Args = append(metadata.Args, c.Arg(i))
			}
			for path, wasmPath := range flags.Files {
				b, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				metadata.Files = append(
					metadata.Files,
					&wasmpluginv1.File{
						Path:     wasmPath,
						Contents: b,
					},
				)
			}
			if proto.Equal(&wasmpluginv1.ExecConfig{}, metadata) {
				if _, err := os.Stderr.WriteString("Nothing to encode\n"); err != nil {
					return err
				}
				return nil
			}
			metadataSection, err := bufwasm.EncodeBufSection(metadata)
			if err != nil {
				return err
			}
			if _, err := os.Stdout.Write(metadataSection); err != nil {
				return err
			}
			return nil
		}),
	}
}

func newExecuteCmd(builder appflag.Builder) *appcmd.Command {
	return &appcmd.Command{
		Use:   "execute",
		Short: "Execute a wasm file with buf extensions",
		Args:  cobra.ExactArgs(1),
		Run: builder.NewRunFunc(func(ctx context.Context, c appflag.Container) error {
			pluginExecutor, err := bufwasm.NewPluginExecutor(".build")
			if err != nil {
				return err
			}
			bin, err := os.ReadFile(c.Arg(0))
			if err != nil {
				return err
			}
			plugin, err := pluginExecutor.CompilePlugin(ctx, bin)
			if err != nil {
				return err
			}
			if err := pluginExecutor.Run(ctx, plugin, os.Stdin, os.Stdout); err != nil {
				if pluginErr := new(bufwasm.PluginExecutionError); errors.As(err, &pluginErr) {
					_, _ = os.Stderr.WriteString(pluginErr.Stderr)
				}
				return err
			}
			return nil
		}),
	}
}

func newInspectCmd(builder appflag.Builder) *appcmd.Command {
	return &appcmd.Command{
		Use:   "inspect",
		Short: "Print buf extensions from wasm file",
		Args:  cobra.ExactArgs(1),
		Run: builder.NewRunFunc(func(ctx context.Context, c appflag.Container) error {
			pluginExecutor, err := bufwasm.NewPluginExecutor(".build")
			if err != nil {
				return err
			}
			bin, err := os.ReadFile(c.Arg(0))
			if err != nil {
				return err
			}
			plugin, err := pluginExecutor.CompilePlugin(ctx, bin)
			if err != nil {
				return err
			}
			if plugin.ExecConfig == nil {
				return fmt.Errorf("no custom section named %q", bufwasm.CustomSectionName)
			}
			// omit file contents for printing
			for _, f := range plugin.ExecConfig.Files {
				f.Contents = nil
			}
			if _, err := os.Stdout.WriteString(protojson.Format(plugin.ExecConfig)); err != nil {
				return err
			}
			return nil
		}),
	}
}

func newNameCmd(builder appflag.Builder) *appcmd.Command {
	return &appcmd.Command{
		Use:   "name",
		Short: "Add name section to wasm file",
		Args:  cobra.ExactArgs(1),
		Run: builder.NewRunFunc(func(ctx context.Context, c appflag.Container) error {
			lengthPrefix := func(b []byte) []byte {
				return append(binary.AppendUvarint(nil, uint64(len(b))), b...)
			}
			// https://webassembly.github.io/spec/core/appendix/custom.html#name-section
			subsecModuleName := append(
				[]byte{0x00}, // name section. module name subsection
				lengthPrefix(
					lengthPrefix([]byte(c.Arg(0))),
				)...,
			)
			customSectionBytes := append(
				lengthPrefix([]byte("name")),
				subsecModuleName...,
			)
			// wasm custom section id
			if _, err := os.Stdout.Write([]byte{0x00}); err != nil {
				return err
			}
			if _, err := os.Stdout.Write(lengthPrefix(customSectionBytes)); err != nil {
				return err
			}
			return nil
		}),
	}
}
