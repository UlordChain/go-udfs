package commands

import (
	"context"
	"fmt"
	"github.com/ipfs/go-ipfs/core/commands/sms"
	"github.com/pkg/errors"
	"gx/ipfs/QmT3rzed1ppXefourpmoZ7tyVQfsGPQZ1pHDngLmCvXxd3/go-path"
	"io"
	"os"
	"time"

	"github.com/ipfs/go-ipfs/core/commands/cmdenv"
	"github.com/ipfs/go-ipfs/core/coreapi/interface"

	cmds "gx/ipfs/QmSXUokcP4TJpFfqozT69AVAYRtzXVMUjzQVkYX41R9Svs/go-ipfs-cmds"
	"gx/ipfs/Qmde5VP1qUkyQXKCfmEUA7bP64V2HAptbJ7phuPp7jXWwg/go-ipfs-cmdkit"
)

const (
	progressBarMinSize = 1024 * 1024 * 8 // show progress bar for outputs > 8MiB
	offsetOptionName   = "offset"
	lengthOptionName   = "length"
)

var CatCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline:          "Show IPFS object data.",
		ShortDescription: "Displays the data contained by an IPFS or IPNS object(s) at the given path.",
	},

	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("ipfs-path", true, true, "The path to the IPFS object(s) to be outputted.").EnableStdin(),
	},
	Options: []cmdkit.Option{
		cmdkit.Int64Option(offsetOptionName, "o", "Byte offset to begin reading from."),
		cmdkit.Int64Option(lengthOptionName, "l", "Maximum number of bytes to read."),
		cmdkit.StringOption(accountOptionName, "Account of user to check"),
		cmdkit.StringOption(tokenOptionName, "The token value for verify"),
	},
	Run: func(req *cmds.Request, res cmds.ResponseEmitter, env cmds.Environment) error {
		if len(req.Arguments) > 1 {
			return errors.New("Do not allow multiple files to be got now")
		}

		// verify token
		tokenInf := req.Options[tokenOptionName]
		if tokenInf == nil {
			return errors.New("must set option token.")
		}
		token := tokenInf.(string)
		p := path.Path(req.Arguments[0])

		err := sms.Get(token, p.String())
		if err != nil {
			return err
		}

		node, err := cmdenv.GetNode(env)
		if err != nil {
			return err
		}

		cfg, _ := node.Repo.Config()
		var(
			account string
			check string
		)
		if cfg.UOSCheck.Enable {
			acc := req.Options[accountOptionName]
			if acc == nil {
				return errors.New("must set option account.")
			}
			account = acc.(string)

			h := req.Options[checkOptionName]
			if h == nil {
				return errors.New("must set option check.")
			}
			check = h.(string)

			_, err = ValidOnUOS(&cfg.UOSCheck, account, check)
			if err != nil {
				return errors.Wrap(err, "valid failed")
			}
		}

		api, err := cmdenv.GetApi(env)
		if err != nil {
			return err
		}

		if !node.OnlineMode() {
			if err := node.SetupOfflineRouting(); err != nil {
				return err
			}
		}

		offset, _ := req.Options[offsetOptionName].(int64)
		if offset < 0 {
			return fmt.Errorf("cannot specify negative offset")
		}

		max, found := req.Options[lengthOptionName].(int64)
		if err != nil {
			return err
		}
		if max < 0 {
			return fmt.Errorf("cannot specify negative length")
		}
		if !found {
			max = -1
		}

		err = req.ParseBodyArgs()
		if err != nil {
			return err
		}

		readers, length, err := cat(req.Context, api, req.Arguments, int64(offset), int64(max))
		if err != nil {
			return err
		}

		/*
			if err := corerepo.ConditionalGC(req.Context, node, length); err != nil {
				re.SetError(err, cmdkit.ErrNormal)
				return
			}
		*/

		res.SetLength(length)
		reader := io.MultiReader(readers...)

		// Since the reader returns the error that a block is missing, and that error is
		// returned from io.Copy inside Emit, we need to take Emit errors and send
		// them to the client. Usually we don't do that because it means the connection
		// is broken or we supplied an illegal argument etc.
		return res.Emit(reader)
	},
	PostRun: cmds.PostRunMap{
		cmds.CLI: func(res cmds.Response, re cmds.ResponseEmitter) error {
			if res.Length() > 0 && res.Length() < progressBarMinSize {
				return cmds.Copy(re, res)
			}

			for {
				v, err := res.Next()
				if err != nil {
					if err == io.EOF {
						return nil
					}
					return err
				}

				switch val := v.(type) {
				case io.Reader:
					bar, reader := progressBarForReader(os.Stderr, val, int64(res.Length()))
					bar.Start()

					err = re.Emit(reader)
					if err != nil {
						return err
					}
				default:
					log.Warningf("cat postrun: received unexpected type %T", val)
				}
			}
		},
	},
}

func cat(ctx context.Context, api iface.CoreAPI, paths []string, offset int64, max int64) ([]io.Reader, uint64, error) {
	readers := make([]io.Reader, 0, len(paths))
	length := uint64(0)
	if max == 0 {
		return nil, 0, nil
	}
	for _, p := range paths {
		fpath, err := iface.ParsePath(p)
		if err != nil {
			return nil, 0, err
		}

		tmCtx, _ := context.WithTimeout(ctx, 2*time.Minute)
		file, err := api.Unixfs().Get(tmCtx, fpath)
		if err != nil {
			return nil, 0, err
		}

		if file.IsDirectory() {
			return nil, 0, iface.ErrIsDir
		}

		fsize, err := file.Size()
		if err != nil {
			return nil, 0, err
		}

		if offset > fsize {
			offset = offset - fsize
			continue
		}

		count, err := file.Seek(offset, io.SeekStart)
		if err != nil {
			return nil, 0, err
		}
		offset = 0

		fsize, err = file.Size()
		if err != nil {
			return nil, 0, err
		}

		size := uint64(fsize - count)
		length += size
		if max > 0 && length >= uint64(max) {
			var r io.Reader = file
			if overshoot := int64(length - uint64(max)); overshoot != 0 {
				r = io.LimitReader(file, int64(size)-overshoot)
				length = uint64(max)
			}
			readers = append(readers, r)
			break
		}
		readers = append(readers, file)
	}
	return readers, length, nil
}
