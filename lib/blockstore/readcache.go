package blockstore

import (
	"context"

	block "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
)

type ReadCache struct {
	cache, bs Blockstore
}

func NewReadCache(base, cache Blockstore) Blockstore {
	// Wrap this in an ID blockstore to avoid caching blocks inlined into
	// CIDs.
	return WrapIDStore(&ReadCache{
		cache: cache,
		bs:    base,
	})
}

var _ (Blockstore) = &ReadCache{}

func (bs *ReadCache) AllKeysChan(ctx context.Context) (<-chan cid.Cid, error) {
	return bs.bs.AllKeysChan(ctx)
}

func (bs *ReadCache) DeleteBlock(c cid.Cid) error {
	_ = bs.cache.DeleteBlock(c)

	return bs.bs.DeleteBlock(c)
}

func (bs *ReadCache) Get(c cid.Cid) (block.Block, error) {
	b, err := bs.cache.Get(c)
	if err == ErrNotFound {
		b, err = bs.bs.Get(c)
		if err != nil {
			return nil, err
		}
		_ = bs.cache.Put(b)
	}
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (bs *ReadCache) GetSize(c cid.Cid) (int, error) {
	size, err := bs.cache.GetSize(c)
	if err == ErrNotFound {
		size, err = bs.bs.GetSize(c)
	}
	return size, err
}

func (bs *ReadCache) Put(blk block.Block) error {
	if has, err := bs.cache.Has(blk.Cid()); err == nil && has {
		return nil
	}
	_ = bs.cache.Put(blk)
	return bs.bs.Put(blk)
}

func (bs *ReadCache) Has(c cid.Cid) (bool, error) {
	if has, err := bs.cache.Has(c); err == nil && has {
		return true, nil
	}
	return bs.bs.Has(c)
}

func (bs *ReadCache) HashOnRead(hor bool) {
	bs.bs.HashOnRead(hor)
}

func (bs *ReadCache) PutMany(blks []block.Block) error {
	newBlks := make([]block.Block, 0, len(blks))
	for _, blk := range blks {
		if has, err := bs.cache.Has(blk.Cid()); err == nil && has {
			continue
		}
		newBlks = append(newBlks, blk)
	}
	_ = bs.cache.PutMany(newBlks)
	return bs.bs.PutMany(newBlks)
}
