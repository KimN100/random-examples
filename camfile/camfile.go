/*
Proof of concept for content addressable memory.
See for example:

	https://en.wikipedia.org/wiki/Venti
	https://en.wikipedia.org/wiki/Merkle_tree

*/
package camfile

import (
	"crypto/md5"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const (
	state_none = iota
	state_open = iota
	state_closed = iota
	state_last = iota

/*
BUGS:
1.  These should be parameters on the server.
*/
	cam_block_size = 1024
	cam_header_size = 32
	cam_indirect_cnt = (cam_block_size - cam_header_size) / 32
)

var (
	state_name []string = []string{
		"state_none",
		"state_open",
		"state_closed",
		"state_last",
	}
)

func stateString(state int) (str string) {

	if state < state_none || state >= state_last {
		// BUG: should probably just panic
		str = fmt.Sprintf("bad state: %d", state)
	} else {
		str = state_name[state]
	}

	return
}

type Server struct {
	client *http.Client
	fsroot string
	state int
}

type Reader struct {
	server *Server
	state int
	ids []string
}

type Writer struct {
	server *Server
	state int
	ids []string
}

/*
conn specifies the root of a file system on localhost,
or the URL of a networked Cam block server.
*/
func NewServer(conn string) (cs *Server, err error) {

	var (
		hroot *http.Client
		fi os.FileInfo
		fsroot string
	)
	if conn == "" {
		err = fmt.Errorf("missing connection string")
		goto out
	}
	if strings.HasPrefix(conn, "http") {
		hroot = &http.Client{}
	} else {
		if fi, err = os.Stat(conn); err != nil {
			err = fmt.Errorf("bad root: %s, %s", conn, err.Error())
			goto out
		} else if !fi.IsDir() {
			err = fmt.Errorf("bad root: %s, %s", conn, err.Error())
			goto out
		} else {
			fsroot = conn
		}
	}

	cs = &Server{ client: hroot, fsroot: fsroot, state: state_open }

out:
	return 
}

/*
Create a resource for managing the copy from a Server to the local writer.
The local writer must be a type that supports Write, for example an *os.File 
or a *bytes.Buffer.  The writer must be created separately.
*/
func (cs *Server) Open(id string) (cr *Reader, err error) {

	if len(id) != 32 {
		return nil, fmt.Errorf("not a block id: " + id)
	}

	cr = &Reader{ server: cs, state: state_open }
	cr.ids = append(cr.ids, id)

	return
}

/*
Create a resource for managing the copy from the local reader to a Server.
The local reader must be a type that supports Read, for example an *os.File 
or a *bytes.Buffer.  The reader must be created separately.
*/
func (cs *Server) Create() (cr *Writer, err error) {
	return &Writer{ server: cs, state: state_open }, nil
}

/*
TODO:
Probably a leak here (any response body?)
*/
func (cs *Server) Close() (err error) {
	if cs.state != state_open {
		panic("unexpected state")
	}
	cs.client = nil
	cs.state = state_closed
	return
}

/*
The Reader is initialized with the root block in Open().
*/
func (cr *Reader) Copy(dst io.Writer) (nn int, err error) {

	if cr.server.state != state_open || cr.state != state_open {
		err = fmt.Errorf("not opened: server %s, reader %s", stateString(cr.server.state), stateString(cr.state))
	} else {
		nn, err = cr.copyToDst(dst)
	}

	return
}

func (cr *Reader) Close() (err error) {
	if cr.state != state_open {
		panic("unexpected state")
	}
	cr.server = nil
	cr.state = state_closed
	cr.ids = nil
	return
}

func (cw *Writer) Copy(src io.Reader) (id string, nn int, err error) {

	if cw.server.state != state_open || cw.state != state_open {
		err = fmt.Errorf("not opened: server %s, reader %s", stateString(cw.server.state), stateString(cw.state))
	} else {
		nn, err = cw.copySrc(src)
		if len(cw.ids) > 1 {
			id, err = cw.copyIds(src)
		} else if len(cw.ids) == 1 {
			id = cw.ids[0]
		} else {
			panic("no ids")
		}
	}

	return
}

func (cw *Writer) Close() (err error) {
	if cw.state != state_open {
		panic("unexpected state")
	}
	cw.server = nil
	cw.state = state_closed
	cw.ids = nil
	return
}

func (cr *Reader) copyToDst(dst io.Writer) (nn int, err error) {
	var (
		id, tag string
		cnt, ii int
		buff [cam_block_size]byte
	)

loop:
	for len(cr.ids) > 0 {
		id, cr.ids = cr.ids[0], cr.ids[1:]
		if err = cr.server.getBlock(id, buff[:]); err != nil {
			if err == io.EOF {
				err = nil
			}
			break loop
		}
		if tag, cnt, err = cr.parseHeader(buff[:]); err != nil {
			break loop
		}
		switch tag {
		case "DATA":
			if cnt, err = dst.Write(buff[cam_header_size:cam_header_size+cnt]); err != nil {
				break loop
			}
			nn += int(cnt)
		case "INDB":
			for ii = cam_header_size; ii < cam_header_size+cnt; ii += 32 {
				cr.ids = append(cr.ids, string(buff[ii:ii+32]))
			}
		default:
			err = fmt.Errorf("unimplemented block type: %s", tag)
			break loop
		}
	}

	return
}

/*
The first 32 bytes describe the block.
*/
func (cr *Reader) parseHeader(data []byte) (blocktype string, blocksize int, err error) {

	var nn int64

	if len(data) != cam_block_size {
		err = fmt.Errorf("BUG: parseHeader: incorrect block size: %d\n", len(data))
		return
	}

	blocktype = string(data[4:8])
	
	if nn, err = strconv.ParseInt(string(data[8:12]), 16, 16); err != nil {
		err = fmt.Errorf("ERROR: parseHeader: failed to convert blocksize: %s, %s\n", data[8:12], err.Error())
		return
	}

	blocksize = int(nn)

	return
}

func (cw *Writer) copySrc(src io.Reader) (nn int, err error) {
	var (
		buff [cam_block_size - cam_header_size]byte
		head, data []byte
		id string
		cnt, salt int
	)

loop:
	for {
		if cnt, err = src.Read(buff[:]); err != nil {
			if err == io.EOF {
				err = nil
			}
			break loop
		}
		salt = 0
		head = []byte(fmt.Sprintf("%04xDATA%04x--------------------", salt, cnt))
		data = buff[:cnt]
		data = append(data, []byte(strings.Repeat("-", cam_block_size - cam_header_size - cnt))...)

		if id, err = cw.server.putBlock(head, data); err != nil {
			break loop
		}

		cw.ids = append(cw.ids, id)
		nn += cnt
	}

	return
}

func (cw *Writer) copyIds(src io.Reader) (id string, err error) {
	var (
		data [cam_block_size - cam_header_size]byte
		head []byte
		cnt, ii, salt int
		newids []string
	)

loop:
	for len(cw.ids) > 0 {
		newids = nil
		for len(cw.ids) > 0 {
			cnt = len(cw.ids)
			if cnt > cam_indirect_cnt {
				cnt = cam_indirect_cnt
			}
			for ii = 0; ii < cnt; ii++ {
				id, cw.ids = cw.ids[0], cw.ids[1:]
				copy(data[ii*32:ii*32+32], id)
			}
			for ; ii < cam_indirect_cnt; ii++ {
				copy(data[ii*32:ii*32+32], "--------------------------------")
			}
			salt = 0
			head = []byte(fmt.Sprintf("%04xINDB%04x--------------------", salt, cnt*32))
			if id, err = cw.server.putBlock(head, data[:]); err != nil {
				break loop
			}

			newids = append(newids, id)
		}
		// assert len(cw.ids) == 0

		if len(newids) > 1 {
			cw.ids = newids
		} else if len(newids) == 1 {
			id = newids[0]
		}
	}

	return}

/*
BUGS:
1.  This should probably be implemented by a Server interface and concrete 
    types that support http and file system cams.
*/
func (cs *Server) putBlock(head, data []byte) (id string, err error) {
/*
Put a block to either the local file system or an http block server.
Return the block id.

*/

	if cs.client != nil {
		panic("unimplemented")
	} else if cs.fsroot != "" {
		id, err = cs.putBlockFile(head, data)
	} else {
		panic("bad Server root")
	}

	return
}

func (cs *Server) putBlockFile(head, data []byte) (id string, err error) {
	var (
		hh hash.Hash
		fh *os.File
		fn string
	)

	hh = md5.New()
	hh.Write(head)
	hh.Write(data)
	id = fmt.Sprintf("%x", hh.Sum(nil))
	fn = cs.fsroot + "/" + id
	if _, err = os.Stat(fn); err != nil {
		if fh, err = os.Create(fn); err == nil {
			fh.Write(head)
			fh.Write(data)
			fh.Close()
		}
	}

	return
}

func (cs *Server) getBlock(id string, data []byte) (err error) {
/*
Get a block from either a local file system or an http block server.
*/
	if cs.client != nil {
		panic("unimplemented")
	} else if cs.fsroot != "" {
		err = cs.getBlockFile(id, data)
	} else {
		panic("bad server root")
	}

	return
}

func (cs *Server) getBlockFile(id string, data []byte) (err error) {
	var (
		fh *os.File
		fn string
		nn int
	)

	fn = cs.fsroot + "/" + id
	if fh, err = os.Open(fn); err != nil {
		goto out
	}
	defer fh.Close()
	if nn, err = io.ReadFull(fh, data); err != nil {
		goto out
	}
	// TODO: check nn
	nn = nn

out:
	return
}
