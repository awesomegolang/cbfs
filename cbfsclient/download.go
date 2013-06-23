package main

import (
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/couchbaselabs/cbfs/client"
	"github.com/dustin/go-humanize"
)

var dlFlags = flag.NewFlagSet("download", flag.ExitOnError)
var dlVerbose = dlFlags.Bool("v", false, "Verbose download")

func saveDownload(filenames []string, oid string, r io.Reader) error {
	ws := []io.Writer{}
	for _, fn := range filenames {
		f, err := os.Create(fn)
		if err != nil {
			os.MkdirAll(filepath.Dir(fn), 0777)
			f, err = os.Create(fn)
		}
		if err != nil {
			return err
		}
		defer f.Close()
		ws = append(ws, f)
	}
	w := io.MultiWriter(ws...)
	n, err := io.Copy(w, r)
	if err == nil {
		verbose(*dlVerbose, "Downloaded %s into %v",
			humanize.Bytes(uint64(n)), strings.Join(filenames, ", "))
	} else {
		log.Printf("Error downloading %v (for %v): %v",
			oid, filenames, err)
	}

	return err
}

func downloadCommand(u string, args []string) {
	dlFlags.Parse(args)

	if dlFlags.NArg() < 2 {
		log.Fatalf("src and dest required")
	}

	destbase := dlFlags.Arg(1)

	client, err := cbfsclient.New(u)
	maybeFatal(err, "Can't build a client: %v", err)

	u = relativeUrl(u, dlFlags.Arg(0))
	log.Printf("Listing from %v with %v", u, client)

	things, err := cbfsclient.List(u)
	maybeFatal(err, "Can't list things: %v", err)

	oids := []string{}
	dests := map[string][]string{}
	for fn, inf := range things.Files {
		dests[inf.OID] = append(dests[inf.OID],
			filepath.Join(destbase, fn))
		oids = append(oids, inf.OID)
	}

	err = client.GetBlobs(4, func(oid string, r io.Reader) error {
		return saveDownload(dests[oid], oid, r)
	}, oids...)

	maybeFatal(err, "Error getting blobs: %v", err)
}