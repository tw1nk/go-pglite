package gopglite

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
)

type PGLite struct {
	wazeroRuntime wazero.Runtime
	wazeroCtx     context.Context
	quitChan      chan struct{}

	pglite api.Module
}

func NewPGLite() *PGLite {
	ctx := context.Background()
	runtime := wazero.NewRuntime(ctx)

	wasi_snapshot_preview1.MustInstantiate(ctx, runtime)

	return &PGLite{
		wazeroRuntime: runtime,
		wazeroCtx:     ctx,
		quitChan:      make(chan struct{}, 1),
	}
}

func (pg *PGLite) Close() error {
	return pg.wazeroRuntime.Close(pg.wazeroCtx)
}

func (pg *PGLite) Start(cfg config) (<-chan struct{}, error) {
	// setup env
	blob, err := setupEnv(cfg)
	if err != nil {
		return nil, err
	}

	pglite, err := pg.wazeroRuntime.InstantiateWithConfig(
		context.Background(),
		blob,
		cfg.ModuleConfig().
			WithArgs("--single", "postgres").
			WithEnv("ENVIRONMENT", "wasi-embed").
			WithEnv("REPL", "Y").
			WithEnv("PGUSER", cfg.pguser).
			WithEnv("PGDATABASE", cfg.pgdatabase),
	)

	if err != nil {
		// Note: Most compilers do not exit the module after running "_start",
		// unless there was an error. This allows you to call exported functions.
		if exitErr, ok := err.(*sys.ExitError); ok && exitErr.ExitCode() != 0 {
			fmt.Fprintf(os.Stderr, "exit_code: %d\n", exitErr.ExitCode())
		} else if !ok {
			return nil, err
		}
	}

	_, err = pglite.ExportedFunction("pg_initdb").Call(context.Background())
	if err != nil {
		return nil, err
	}

	pg.pglite = pglite

	// write to weird socket files
	_, err = pglite.ExportedFunction("use_socketfile").Call(context.Background())
	if err != nil {
		return nil, err
	}

	if cfg.listen {
		// start listener.
		// This doesn't work right now as pglite for some
		// reason doesn't speak the postgres protocol.
		go pg.networkListener(cfg.listenPort)
	}

	out := make(chan struct{})
	go func() {
		out <- struct{}{}
		<-pg.quitChan

		// todo: close listener if it's running

		err = pglite.Close(context.Background())
		if err != nil {
			panic(err)
		}
	}()

	return out, nil
}

func (pg *PGLite) networkListener(port int) {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		fmt.Println(err)
		return
	}

	for {
		// Accept connection on port
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}

		// Handle connection
		go pg.handleConnection(conn)
	}
}

func (pg *PGLite) handleConnection(conn net.Conn) {
	// Make a buffer to hold incoming data.
	buf := make([]byte, 1024)
	// Read the incoming connection into the buffer.
	reqLen, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading:", err.Error())
	}

	// send to pglite
	resp, err := pg.ExecProtocol(buf[:reqLen])
	if err != nil {
		fmt.Println("Error executing protocol:", err.Error())
	}

	// Send a response back to person contacting us.
	conn.Write(resp)
	// Close the connection when you're done with it.
	conn.Close()
}

func (pg *PGLite) Exec(query string) ([]byte, error) {
	return pg.ExecProtocol(append([]byte(query), 0))
}

func (pg *PGLite) ExecProtocol(message []byte) ([]byte, error) {
	if message[len(message)-1] != 0 {
		message = append(message, 0)
	}

	msgLen := len(message)
	pg.pglite.Memory().Write(1, message)

	// No idea why this doesn't work
	/*
		_, err := pg.pglite.ExportedFunction("interactive_write").
				Call(context.Background(), uint64(msgLen))

			if err != nil {
				return nil, err
			}
	*/

	_, err := pg.pglite.ExportedFunction("interactive_one").Call(context.Background())
	if err != nil {
		return nil, err
	}

	// read the response
	msgStart := msgLen + 2
	retMsgLen, err := pg.pglite.ExportedFunction("interactive_read").Call(context.Background())
	if err != nil {
		return nil, err
	}

	data, ok := pg.pglite.Memory().Read(uint32(msgStart), uint32(retMsgLen[0]))
	if !ok {
		return nil, fmt.Errorf("could not read response")
	}

	return data, nil
}
