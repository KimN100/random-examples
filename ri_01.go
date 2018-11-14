/*
Proof of concept of recursive quad tree for spatial data.

Nodes come in two types:
	Inner, holds subtrees only, no points
	Leaf, holds points only, no sub trees.
*/

package main

import (
	"fmt"
)

type Point struct {
	xx, yy int
}

type Node interface {
	Add(pt *Point) bool
	Find(pt *Point) *Point
	Contains(pt *Point) bool
	Split() Node
	Dump()
}

const (
	inner_nw = 0
	inner_ne = 1
	inner_sw = 2
	inner_se = 3
)

const inner_quads = 4

type Inner struct {
	xmin, ymin, xmax, ymax int
	quads [inner_quads]Node
}

const leaf_minsize = 10 	// minimum bucket size
const leaf_minextent = 10 	// minimum width and height

type Leaf struct {
	xmin, ymin, xmax, ymax int
	size, full int
	pnts []*Point
}

func NewPoint(xx, yy int) *Point {
	pt := new(Point)
	pt.xx = xx
	pt.yy = yy
	return pt
}

var root Node

func NewLeaf(left, top, width, height int) *Leaf {

	lf := new(Leaf)

	lf.xmin = left
	lf.ymin = top
	lf.xmax = left + width
	lf.ymax = top + height

	lf.pnts = make([]*Point, leaf_minsize)
	lf.size = leaf_minsize
	lf.full = 0

	return lf
}

func NewInner(left, top, width, height int) *Inner {

	in := new(Inner)

	in.xmin = left
	in.ymin = top
	in.xmax = left + width
	in.ymax = top + height

	return in
}

func (in *Inner) Contains(pt *Point) bool {
	if (in.xmin <= pt.xx && pt.xx < in.xmax) {
		if (in.ymin <= pt.yy && pt.yy < in.ymax) {
			return true
		}
	}
	return false
}

func (in *Inner) Add(pt *Point) bool {

	var ok bool

	ii := 0
	for ; ii < inner_quads; ii++ {
		if in.quads[ii].Contains(pt) {
			if ok = in.quads[ii].Add(pt); !ok {
				// Must be a full leaf
				in.quads[ii] = in.quads[ii].Split() 	// should orphan the old leaf for GC
				if ok = in.quads[ii].Add(pt); !ok {
					// should be impossible
					panic("BUG: double fail to Add")
				}
			}
			break
		}
	}

	if ii == inner_quads {
		root.Dump()
		panic("BUG: point out of bounds\n")
	}

	return ok
}

func (in *Inner) Find(pt *Point) *Point {
	return nil
}

func (in *Inner) Dump() {

	fmt.Printf("Inner @ %p\n", in)
	fmt.Printf("\t(%d, %d, %d, %d)\n", in.xmin, in.ymin, in.xmax, in.ymax)
	for ii := 0; ii < inner_quads; ii++ {
		fmt.Printf("\t%2d: %p\n", ii, in.quads[ii])
	}
	for ii := 0; ii < inner_quads; ii++ {
		in.quads[ii].Dump()
	}
}

func (lf *Leaf) Contains(pt *Point) bool {

	if (lf.xmin <= pt.xx && pt.xx < lf.xmax) {
		if (lf.ymin <= pt.yy && pt.yy < lf.ymax) {
			return true
		}
	}

	return false
}

func (lf *Leaf) Add(pt *Point) bool {
	if !lf.Contains(pt) {
		panic(fmt.Sprintf("BUG: point (%d, %d) outside leaf (%d, %d, %d, %d)\n", pt.xx, pt.yy, lf.xmin, lf.ymin, lf.xmax, lf.ymax));
	}

	if lf.full < lf.size {
		lf.pnts[lf.full] = pt
		lf.full++
		return true
	} else {
		return false
	}
}
func (lf *Leaf) Find(pt *Point) (*Point) {
	
	var needle *Point

	for ii := 0; ii < lf.full; ii++ {
		if pt.xx == lf.pnts[ii].xx && pt.yy == lf.pnts[ii].yy {
			needle = lf.pnts[ii]
			break
		}
	}

	return needle 	// point or nil
}

func (lf *Leaf) Dump() {
	fmt.Printf("Leaf @ %p\n", lf)
	fmt.Printf("\t(%d, %d, %d, %d)\n", lf.xmin, lf.ymin, lf.xmax, lf.ymax)
	fmt.Printf("\tfull: %d\n", lf.full)
	for ii := 0; ii < lf.full; ii++ {
		fmt.Printf("\t%2d: %+v\n", ii, lf.pnts[ii])
	}
}

func (in *Inner) Split() Node {
	panic("BUG: Split should never be called on Inner")
}

func (lf *Leaf) Centroid() (int, int) {

	xc := 0
	yc := 0

	for ii := 0; ii < lf.full; ii++ {
		xc += lf.pnts[ii].xx
		yc += lf.pnts[ii].yy
	}

	xc = xc / lf.full
	yc = yc / lf.full

	return xc, yc
}

func (lf *Leaf) Grow() {
	lf.Resize(lf.size * 3 / 2)
}

func (lf *Leaf) Resize(newsize int) {
	if newsize < leaf_minsize {
		newsize = leaf_minsize
	}

	newpnts := make([]*Point, newsize)

	if lf.full > 0 {
		if lf.pnts == nil {
			panic("BUG: no points")
		}
		copy(newpnts, lf.pnts)
	}

	lf.pnts = newpnts 	// should orphan the old slice for GC
	lf.size = newsize
}

func (lf *Leaf) Split() Node {

	xc, yc := lf.Centroid()

	if xc - lf.xmin < leaf_minextent || lf.xmax - xc < leaf_minextent || yc - lf.ymin < leaf_minextent || lf.ymax - yc < leaf_minextent {
		// New split would be too small, so just grow the bucket
		lf.Grow()
		return lf
	}

	in := NewInner(lf.xmin, lf.ymin, lf.xmax - lf.xmin, lf.ymax - lf.ymin) 	// creates a bunch of new leafs too.

	in.quads[inner_nw] = NewLeaf(lf.xmin, lf.ymin, xc - lf.xmin, yc - lf.ymin)
	in.quads[inner_ne] = NewLeaf(xc, lf.ymin, lf.xmax - xc, yc - lf.ymin)
	in.quads[inner_sw] = NewLeaf(lf.xmin, yc, xc - lf.xmin, lf.ymax - yc)
	in.quads[inner_se] = NewLeaf(xc, yc, lf.xmax - xc, lf.ymax - yc)

	for ii := 0; ii < lf.full; ii++ {
		in.Add(lf.pnts[ii])
	}

	return in
}

func init() {
	root = NewLeaf(0, 0, 1000, 1000)
}

func addpt(pt *Point) {
	if !root.Add(pt) {
		// Only a leaf can return false, so we know to split, and create a new Inner root.
		root = root.Split() 	// should orphan the old leaf for GC
		if !root.Add(pt) {
			// should be impossible
			panic("BUG: double fail to Add")
		}
	}
}

func main() {

	var pt *Point

	for ii := 100; ii < 901; ii += 100 {
		pt = NewPoint(ii, ii)
		addpt(pt)
	}

	root.Dump()
}
