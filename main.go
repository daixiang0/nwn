package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/scanner"
	"image"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
)

var (
	doWrite = flag.Bool("w", false, "doWrite result to (source) file instead of stdout")
	doDiff  = flag.Bool("d", false, "display diffs instead of rewriting files")

	whiteNoise = regexp.MustCompile("[ \t]*\n")

	exitCode = 0
)

func report(err error) {
	scanner.PrintError(os.Stderr, err)
	exitCode = 2
}

func parseFlags() []string {
	flag.Parse()
	return flag.Args()
}

func usage() {
	_, _ = fmt.Fprintf(os.Stderr, "usage: nwn [flags] [path ...]\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func diff(b1, b2 []byte, filename string) (data []byte, err error) {
	f1, err := writeTempFile("", "nwn", b1)
	if err != nil {
		return
	}
	defer os.Remove(f1)

	f2, err := writeTempFile("", "nwn", b2)
	if err != nil {
		return
	}
	defer os.Remove(f2)

	cmd := "diff"

	data, err = exec.Command(cmd, "-u", f1, f2).CombinedOutput()
	if len(data) > 0 {
		// diff exits with a non-zero status when the files don't match.
		// Ignore that failure as long as we get output.
		return replaceTempFilename(data, filename)
	}
	return
}

func writeTempFile(dir, prefix string, data []byte) (string, error) {
	file, err := ioutil.TempFile(dir, prefix)
	if err != nil {
		return "", err
	}
	_, err = file.Write(data)
	if err1 := file.Close(); err == nil {
		err = err1
	}
	if err != nil {
		os.Remove(file.Name())
		return "", err
	}
	return file.Name(), nil
}

// replaceTempFilename replaces temporary filenames in diff with actual one.
//
// --- /tmp/gofmt316145376	2017-02-03 19:13:00.280468375 -0500
// +++ /tmp/gofmt617882815	2017-02-03 19:13:00.280468375 -0500
// ...
// ->
// --- path/to/file.go.orig	2017-02-03 19:13:00.280468375 -0500
// +++ path/to/file.go	2017-02-03 19:13:00.280468375 -0500
// ...
func replaceTempFilename(diff []byte, filename string) ([]byte, error) {
	bs := bytes.SplitN(diff, []byte{'\n'}, 3)
	if len(bs) < 3 {
		return nil, fmt.Errorf("got unexpected diff for %s", filename)
	}
	// Preserve timestamps.
	var t0, t1 []byte
	if i := bytes.LastIndexByte(bs[0], '\t'); i != -1 {
		t0 = bs[0][i:]
	}
	if i := bytes.LastIndexByte(bs[1], '\t'); i != -1 {
		t1 = bs[1][i:]
	}
	// Always print filepath with slash separator.
	f := filepath.ToSlash(filename)
	bs[0] = []byte(fmt.Sprintf("--- %s%s", f+".orig", t0))
	bs[1] = []byte(fmt.Sprintf("+++ %s%s", f, t1))
	return bytes.Join(bs, []byte{'\n'}), nil
}

func processFile(filename string, out io.Writer) error {
	var err error

	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, _, err := image.Decode(f); err == nil {
		fmt.Printf("skip image file %s\n", filename)
		return nil
	}

	src, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	ret := whiteNoise.ReplaceAll(src, []byte{'\n'})

	if !bytes.Equal(src, ret) {
		exitCode = 1

		if *doWrite {
			// On Windows, we need to re-set the permissions from the file. See golang/go#38225.
			var perms os.FileMode
			if fi, err := os.Stat(filename); err == nil {
				perms = fi.Mode() & os.ModePerm
			}
			err = ioutil.WriteFile(filename, ret, perms)
			if err != nil {
				return err
			}
		}
		if *doDiff {
			data, err := diff(src, ret, filename)
			if err != nil {
				return fmt.Errorf("failed to diff: %v", err)
			}
			fmt.Printf("diff -u %s %s\n", filepath.ToSlash(filename+".orig"), filepath.ToSlash(filename))
			if _, err := out.Write(data); err != nil {
				return fmt.Errorf("failed to doWrite: %v", err)
			}
		}
	}
	if !*doWrite && !*doDiff {
		if _, err = out.Write(ret); err != nil {
			return fmt.Errorf("failed to doWrite: %v", err)
		}
	}

	return err

}

func visitFile(path string, f os.FileInfo, err error) error {
	if err == nil {
		err = processFile(path, os.Stdout)
	}
	if err != nil {
		report(err)
	}
	return nil
}

func walkDir(path string) {
	_ = filepath.Walk(path, visitFile)
}

func main() {
	flag.Usage = usage
	paths := parseFlags()
	for _, path := range paths {
		switch dir, err := os.Stat(path); {
		case err != nil:
			report(err)
		case dir.IsDir():
			walkDir(path)
		default:
			if err := processFile(path, os.Stdout); err != nil {
				report(err)
			}
		}
	}
	os.Exit(exitCode)
}
