package camfile

import (
	"crypto/md5"
	"fmt"
	"hash"
	"io"
	"os"
	"testing"
)

const (
	cam_root_local = "/home/mike/Dev/SecureServer/cam"
	cam_root_http = "http://localhost:8080"
)

func TestNewServer(t *testing.T) {
	t.Run("noconn", newServerNoConnection)
	t.Run("http", newServerHttp)
	t.Run("local-badpath", newServerLocalBadPath)
	t.Run("local-goodpath", newServerLocalGoodPath)
}

func TestOpen(t *testing.T) {
	t.Run("local", openLocal)
//	t.Run("http", openHttp)
}

func TestCreate(t *testing.T) {
	t.Run("local", createLocal)
//	t.Run("http", createHttp)
}

func TestWriteToCam(t *testing.T) {
	t.Run("write-local-oneblock", writeToCamOne)
	t.Run("write-local-twoblock", writeToCamTwo)
}

func TestReadFromCam(t *testing.T) {
	t.Run("read-local-oneblock", readFromCamOne)
//	t.Run("read-local-twoblock", readFromCamTwo)
}

/*
TODO: should we allow empty files?
*/
func writeToCamZero(t *testing.T) {
	t.Fatal("unimplemented")
}

/*
Read a file that should have been uploaded in a previous test.
*/
func readFromCamOne(t *testing.T) {

	var (
		cs *Server
		cr *Reader
		err error
		fh *os.File
		id string
		nn int
		hh hash.Hash
		data [32 * 1024]byte 	// large enough to be too large
	)
	if cs, err = NewServer(cam_root_local); err != nil {
		t.Fatal("failed to create server", err.Error())
	}
	defer cs.Close()

	if cr, err = cs.Open("f4b946cdd69b35a8a66fd2b1e56245e9"); err != nil {
		t.Fatal("failed to create reader", err.Error())
	}
	defer cr.Close()

	if fh, err = os.Create("/tmp/camfile-test.dat"); err != nil {
		t.Fatal("failed to create tmp file", err.Error())
	}

	if nn, err = cr.Copy(fh); err != nil {
		t.Fatal("failed to read cam", err.Error())
	}
	
	fh.Close()

	if nn != 993 {
		t.Fatal("unexpected copy size", nn)
	}

	if fh, err = os.Open("/tmp/camfile-test.dat"); err != nil {
		t.Fatal("failed to reopen tmp file", err.Error())
	}

	if nn, err = io.ReadFull(fh, data[:]); err != nil {
		if err != io.ErrUnexpectedEOF {
			t.Fatal("failed to reread tmp file", err.Error())
		}
	}
	fh.Close()

	hh = md5.New()
	hh.Write(data[:nn])
	id = fmt.Sprintf("%x", hh.Sum(nil))

	if id != "60a3f7763758c3b52320d683b89e489f" {
		t.Fatal("unexpected hash", id)
	}

}

func writeToCamOne(t *testing.T) {
	var (
		cs *Server
		cw *Writer
		err error
		fh *os.File
		id string
		nn int
	)

	nn = nn

	if cs, err = NewServer(cam_root_local); err != nil {
		t.Fatal("failed to create server, ", err.Error())
	}
	defer cs.Close()

	if cw, err = cs.Create(); err != nil {
		t.Fatal("failed to create writer, ", err.Error())
	}
	defer cw.Close()

	if fh, err = os.Open("testdata/camfile-test-one-block.dat"); err != nil {
		t.Fatal("failed to open test data, ", err.Error())
	}
	defer fh.Close()

	if id, nn, err = cw.Copy(fh); err != nil {
		t.Fatal("failed to copy to cam, ", err.Error())
	}

	if id != "ae0fab12f558c802558726d04cb5aed6" {
		t.Fatal("unexpected root block", id)
	}
	if nn != 992 {
		t.Fatal("unexpected upload size", nn)
	}
}

func writeToCamTwo(t *testing.T) {
	var (
		cs *Server
		cw *Writer
		err error
		fh *os.File
		id string
		nn int
	)

	nn = nn

	if cs, err = NewServer(cam_root_local); err != nil {
		t.Fatal("failed to create server, ", err.Error())
	}
	defer cs.Close()

	if cw, err = cs.Create(); err != nil {
		t.Fatal("failed to create writer, ", err.Error())
	}
	defer cw.Close()

	if fh, err = os.Open("testdata/camfile-test-two-block.dat"); err != nil {
		t.Fatal("failed to open test data, ", err.Error())
	}
	defer fh.Close()

	if id, nn, err = cw.Copy(fh); err != nil {
		t.Fatal("failed to copy to cam, ", err.Error())
	}

	if id != "f4b946cdd69b35a8a66fd2b1e56245e9" {
		t.Fatal("unexpected root block", id)
	}
	if nn != 993 {
		t.Fatal("unexpected upload size", nn)
	}
}

/*
Read a known file from the Cam.
Ensure the expected number of bytes is reported.
Ensure the file matches a known md5.
*/
/*
func TestReadFromCam(t *testing.T) {
	var (
		cs *Server
		cr *Reader
		err error
		fh *os.File
		nn int
	)

	nn = nn
	
	if cs, err = NewServer(cam_root_local); err != nil {
		t.Fatal("failed to create server, ", err.Error())
	}
	defer cs.Close()

	if cr, err = cs.Open(); err != nil {
		t.Fatal("failed to create reader, ", err.Error())
	}
	defer cr.Close()

	if fh, err = os.Create("/tmp/camfile-test.dat"); err != nil {
		t.Fatal("failed to create tmp file, ", err.Error())
	}
	defer fh.Close()

	if nn, err = cr.Copy(fh); err != nil {
		t.Fatal("failed to copy from cam, ", err.Error())
	}

	t.Fatal("unimplemented copy count test")
	t.Fatal("unimplemented md5 test")
}
*/
// Must have a connection string
func newServerNoConnection(t *testing.T) {
	var (
		cs *Server
		err error
	)

	cs, err = NewServer("")

	if cs != nil {
		t.Error("unexpected server: ", cs)
	}
	if err == nil {
		t.Error("missing error: ")
	}
	if (cs == nil && err == nil) || (cs != nil && err != nil) {
		t.Error("must have either server or error")
	}
	cs = nil
}

// http connections are unimplemented.
func newServerHttp(t *testing.T) {
	var (
		cs *Server
		err error
	)

	cs, err = NewServer("http://foo/bar")
	if cs == nil {
		t.Fatal("expected server")
	}
	if err != nil {
		t.Error("unexpected error: ", err.Error())
	}

	if cs.state != state_open {
		t.Error("incorrect server state, ", cs.state)
	}

	cs.Close()

	if cs.state != state_closed {
		t.Error("incorrect server state, ", cs.state)
	}

	cs = nil
}

func newServerLocalBadPath(t *testing.T) {
	var (
		cs *Server
		err error
	)

	// file system strings must refer to valid directories
	cs, err = NewServer("/tmp/bad/path")
	if cs != nil {
		t.Error("unexpected server: ", cs)
	}
	if err == nil {
		t.Error("expected error")
	}
	if err != nil && err.Error() != "bad root: /tmp/bad/path, stat /tmp/bad/path: no such file or directory" {
		t.Error("expected error message: ", err.Error())
	}
	cs = nil
}

func newServerLocalGoodPath(t *testing.T) {
	var (
		cs *Server
		err error
	)

	cs, err = NewServer(cam_root_local)
	if cs == nil {
		t.Fatal("expected server")
	}
	if err != nil {
		t.Error("unexpected error: ", err.Error())
	}
	cs.Close()

	cs = nil
}

func openLocal(t *testing.T) {
	var (
		cs *Server
		cr *Reader
		err error
	)

	if cs, err = NewServer(cam_root_local); err != nil {
		t.Fatal("failed to create server: ", err.Error())
	}

	if cr, err = cs.Open("bogus 32 char md5-ish string ---"); err != nil {
		t.Fatal("failed to create reader: ", err.Error())
	}

	if cr.server == nil {
		t.Fatal("failed to initialize cr.server (1)")
	}
	if cr.server != cs {
		t.Fatal("failed to initialize cr.server (2)")
	}
	if cr.state != state_open {
		t.Fatal("failed to initialize cr.state")
	}

	if cr != nil {
		cr.Close()
	}
	if cr.state != state_closed {
		t.Fatal("failed to close cr.state")
	}
	cr = nil

	if cs != nil {
		cs.Close()
	}
	cs = nil
}

func openHttp(t *testing.T) {
	var (
		cs *Server
		cr *Reader
		err error
	)

	if cs, err = NewServer(cam_root_http); err != nil {
		t.Fatal("failed to create server: ", err.Error())
	}

	if cr, err = cs.Open("bogus 32 char md5-ish string ---"); err != nil {
		t.Fatal("failed to create reader: ", err.Error())
	}

	if cr.server == nil {
		t.Fatal("failed to initialize cr.server (1)")
	}
	if cr.server != cs {
		t.Fatal("failed to initialize cr.server (2)")
	}
	if cr.state != state_open {
		t.Fatal("failed to initialize cr.state")
	}

	if cr != nil {
		cr.Close()
	}
	if cr.state != state_closed {
		t.Fatal("failed to close cr.state")
	}
	cr = nil

	if cs != nil {
		cs.Close()
	}
	cs = nil
}

func createLocal(t *testing.T) {
	var (
		cs *Server
		cw *Writer
		err error
	)

	if cs, err = NewServer(cam_root_local); err != nil {
		t.Fatal("failed to create server")
	}
	if cw, err = cs.Create(); err != nil {
		t.Fatal("failed to create writer")
	}

	if cw.server == nil {
		t.Fatal("failed to initialize cw.server (1)")
	}
	if cw.server != cs {
		t.Fatal("failed to initialize cw.server (2)")
	}
	if cw.state != state_open {
		t.Fatal("failed to initialize cw.state")
	}

	if cw != nil {
		cw.Close()
	}
	if cw.state != state_closed {
		t.Fatal("failed to close cw.state")
	}
	cw = nil

	if cs != nil {
		cs.Close()
	}
	cs = nil
}

func createHttp(t *testing.T) {
	var (
		cs *Server
		cw *Writer
		err error
	)

	t.Fatal("unimplemented")

	cs = cs
	cw = cw
	err = err
}

