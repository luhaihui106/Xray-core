package limiter

import (
	"context"
	"strings"
	"sync"

	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/buf"
	"golang.org/x/time/rate"
)

const bytesPerMbps = 1_000_000 / 8

type Pair struct {
	Up   *rate.Limiter
	Down *rate.Limiter
}

type entry struct {
	upMbps   uint32
	downMbps uint32
	pair     Pair
}

type Manager struct {
	mu      sync.Mutex
	clients map[string]entry
}

func NewManager() *Manager {
	return &Manager{clients: make(map[string]entry)}
}

func key(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// ForUser returns token buckets shared by all concurrent connections for one client.
func (m *Manager) ForUser(email string, upMbps, downMbps uint32) Pair {
	k := key(email)
	if k == "" || (upMbps == 0 && downMbps == 0) {
		return Pair{}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if old, ok := m.clients[k]; ok && old.upMbps == upMbps && old.downMbps == downMbps {
		return old.pair
	}

	p := Pair{Up: newLimiter(upMbps), Down: newLimiter(downMbps)}
	m.clients[k] = entry{upMbps: upMbps, downMbps: downMbps, pair: p}
	return p
}

func newLimiter(mbps uint32) *rate.Limiter {
	if mbps == 0 {
		return nil
	}
	bytesPerSecond := int64(mbps) * bytesPerMbps
	burst := int(bytesPerSecond / 10)
	if burst < 64*1024 {
		burst = 64 * 1024
	}
	if burst > 4*1024*1024 {
		burst = 4 * 1024 * 1024
	}
	return rate.NewLimiter(rate.Limit(bytesPerSecond), burst)
}

func waitBytes(ctx context.Context, lim *rate.Limiter, n int) error {
	if lim == nil || n <= 0 {
		return nil
	}
	burst := lim.Burst()
	for n > 0 {
		step := n
		if step > burst {
			step = burst
		}
		if err := lim.WaitN(ctx, step); err != nil {
			return err
		}
		n -= step
	}
	return nil
}

type Reader struct {
	ctx context.Context
	r   buf.Reader
	lim *rate.Limiter
}

func WrapReader(ctx context.Context, r buf.Reader, lim *rate.Limiter) buf.Reader {
	if r == nil || lim == nil {
		return r
	}
	return &Reader{ctx: ctx, r: r, lim: lim}
}

func (r *Reader) ReadMultiBuffer() (buf.MultiBuffer, error) {
	mb, err := r.r.ReadMultiBuffer()
	if !mb.IsEmpty() {
		if limitErr := waitBytes(r.ctx, r.lim, int(mb.Len())); limitErr != nil {
			buf.ReleaseMulti(mb)
			return nil, limitErr
		}
	}
	return mb, err
}

func (r *Reader) Interrupt()   { common.Interrupt(r.r) }
func (r *Reader) Close() error { return common.Close(r.r) }

type Writer struct {
	ctx context.Context
	w   buf.Writer
	lim *rate.Limiter
}

func WrapWriter(ctx context.Context, w buf.Writer, lim *rate.Limiter) buf.Writer {
	if w == nil || lim == nil {
		return w
	}
	return &Writer{ctx: ctx, w: w, lim: lim}
}

func (w *Writer) WriteMultiBuffer(mb buf.MultiBuffer) error {
	if !mb.IsEmpty() {
		if err := waitBytes(w.ctx, w.lim, int(mb.Len())); err != nil {
			buf.ReleaseMulti(mb)
			return err
		}
	}
	return w.w.WriteMultiBuffer(mb)
}

func (w *Writer) Interrupt()   { common.Interrupt(w.w) }
func (w *Writer) Close() error { return common.Close(w.w) }
