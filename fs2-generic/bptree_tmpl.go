package main
import "text/template"

var bptreeTmpl = template.Must(template.New("tmpl").Parse(
`package {{.packageName}}

import (
	"sync"
)

import (
	"github.com/timtadh/fs2/bptree"
	"github.com/timtadh/fs2/fmap"
)

import (
	{{range $imp := .imports}}"{{$imp}}"
	{{end}}
)


type MultiMap interface {
	Keys() (KeyIterator, error)
	Values() (ValueIterator, error)
	Iterate() (Iterator, error)
	Find(key {{.keyType}}) (Iterator, error)
	Has(key {{.keyType}}) (bool, error)
	Count(key {{.keyType}}) (int, error)
	Add(key {{.keyType}}, value {{.valueType}}) error
	Remove(key {{.keyType}}, where func({{.valueType}}) bool) error
	Size() int
	Close() error
	Delete() error
}

type Iterator func() ({{.keyType}}, {{.valueType}}, error, Iterator)
type KeyIterator func() ({{.keyType}}, error, KeyIterator)
type ValueIterator func() ({{.valueType}}, error, ValueIterator)

func Do(run func() (Iterator, error), do func(key {{.keyType}}, value {{.valueType}}) error) error {
	kvi, err := run()
	if err != nil {
		return err
	}
	var key {{.keyType}}
	var value {{.valueType}}
	for key, value, err, kvi = kvi(); kvi != nil; key, value, err, kvi = kvi() {
		e := do(key, value)
		if e != nil {
			return e
		}
	}
	return err
}

func DoKey(run func() (KeyIterator, error), do func({{.keyType}}) error) error {
	it, err := run()
	if err != nil {
		return err
	}
	var item {{.keyType}}
	for item, err, it = it(); it != nil; item, err, it = it() {
		e := do(item)
		if e != nil {
			return e
		}
	}
	return err
}

func DoValue(run func() (ValueIterator, error), do func({{.valueType}}) error) error {
	it, err := run()
	if err != nil {
		return err
	}
	var item {{.valueType}}
	for item, err, it = it(); it != nil; item, err, it = it() {
		e := do(item)
		if e != nil {
			return e
		}
	}
	return err
}


type BpTree struct {
	bf *fmap.BlockFile
	bpt *bptree.BpTree
	mutex sync.Mutex
}

func AnonBpTree() (*BpTree, error) {
	bf, err := fmap.Anonymous(fmap.BLOCKSIZE)
	if err != nil {
		return nil, err
	}
	return newBpTree(bf)
}

func NewBpTree(path string) (*BpTree, error) {
	bf, err := fmap.CreateBlockFile(path)
	if err != nil {
		return nil, err
	}
	return newBpTree(bf)
}

func OpenBpTree(path string) (*BpTree, error) {
	bf, err := fmap.OpenBlockFile(path)
	if err != nil {
		return nil, err
	}
	bpt, err := bptree.Open(bf)
	if err != nil {
		return nil, err
	}
	b := &BpTree{
		bf: bf,
		bpt: bpt,
	}
	return b, nil
}

func newBpTree(bf *fmap.BlockFile) (*BpTree, error) {
	bpt, err := bptree.New(bf, {{.keySize}}, {{.valueSize}})
	if err != nil {
		return nil, err
	}
	b := &BpTree{
		bf: bf,
		bpt: bpt,
	}
	return b, nil
}

func (b *BpTree) Close() error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.bf.Close()
}

func (b *BpTree) Delete() error {
	err := b.Close()
	if err != nil {
		return err
	}
	if b.bf.Path() != "" {
		return b.bf.Remove()
	}
	return nil
}

func (b *BpTree) Size() int {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.bpt.Size()
}

func (b *BpTree) Add(key {{.keyType}}, val {{.valueType}}) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.bpt.Add({{.serializeKey}}(key), {{.serializeValue}}(val))
}

func (b *BpTree) Count(key {{.keyType}}) (int, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.bpt.Count({{.serializeKey}}(key))
}

func (b *BpTree) Has(key {{.keyType}}) (bool, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.bpt.Has({{.serializeKey}}(key))
}

func (b *BpTree) kvIter(kvi bptree.KVIterator) (it Iterator) {
	it = func() (key {{.keyType}}, value {{.valueType}}, err error, _ Iterator) {
		b.mutex.Lock()
		defer b.mutex.Unlock()
		var k, v []byte
		k, v, err, kvi = kvi()
		if err != nil {
			return {{.keyEmpty}}, {{.valueEmpty}}, err, nil
		}
		if kvi == nil {
			return {{.keyEmpty}}, {{.valueEmpty}}, nil, nil
		}
		key = {{.deserializeKey}}(k)
		value = {{.deserializeValue}}(v)
		return key, value, nil, it
	}
	return it
}

func (b *BpTree) keyIter(raw bptree.Iterator) (it KeyIterator) {
	it = func() (key {{.keyType}}, err error, _ KeyIterator) {
		b.mutex.Lock()
		defer b.mutex.Unlock()
		var i []byte
		i, err, raw = raw()
		if err != nil {
			return {{.keyEmpty}}, err, nil
		}
		if raw == nil {
			return {{.keyEmpty}}, nil, nil
		}
		key = {{.deserializeKey}}(i)
		return key, nil, it
	}
	return it
}

func (b *BpTree) valueIter(raw bptree.Iterator) (it ValueIterator) {
	it = func() (value {{.valueType}}, err error, _ ValueIterator) {
		b.mutex.Lock()
		defer b.mutex.Unlock()
		var i []byte
		i, err, raw = raw()
		if err != nil {
			return {{.valueEmpty}}, err, nil
		}
		if raw == nil {
			return {{.valueEmpty}}, nil, nil
		}
		value = {{.deserializeValue}}(i)
		return value, nil, it
	}
	return it
}

func (b *BpTree) Keys() (it KeyIterator, err error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	raw, err := b.bpt.Keys()
	if err != nil {
		return nil, err
	}
	return b.keyIter(raw), nil
}

func (b *BpTree) Values() (it ValueIterator, err error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	raw, err := b.bpt.Values()
	if err != nil {
		return nil, err
	}
	return b.valueIter(raw), nil
}

func (b *BpTree) Find(key {{.keyType}}) (it Iterator, err error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	raw, err := b.bpt.Find({{.serializeKey}}(key))
	if err != nil {
		return nil, err
	}
	return b.kvIter(raw), nil
}

func (b *BpTree) Iterate() (it Iterator, err error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	raw, err := b.bpt.Iterate()
	if err != nil {
		return nil, err
	}
	return b.kvIter(raw), nil
}

func (b *BpTree) Remove(key {{.keyType}}, where func({{.valueType}}) bool) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.bpt.Remove({{.serializeKey}}(key), func(bytes []byte) bool {
		return where({{.deserializeValue}}(bytes))
	})
}

`))
