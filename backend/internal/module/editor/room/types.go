package room

import (
	"errors"

	"backend/internal/module/editor/entity"
)

type CmdKind int

const (
	CmdPlace CmdKind = iota
	CmdDelete
	CmdJoin
	CmdLeave
)

type Cmd struct {
	Kind    CmdKind
	CharID  string
	// place:
	Item    *entity.DecorationItem
	X, Y    int
	PlaceID string
	// delete:
	TargetID string
	// join:
	Coins   int
	Reply   chan CmdResult
}

type CmdResult struct {
	Placement *entity.Placement
	NewCoins  int
	Err       error
}

var (
	ErrOccupied          = errors.New("ô đã có vật thể")
	ErrInsufficientCoins = errors.New("không đủ coins để mua vật phẩm này")
	ErrNotOwner          = errors.New("bạn không có quyền xóa vật phẩm này")
	ErrNotFound          = errors.New("vật phẩm không tồn tại hoặc đã bị xóa")
	ErrBusy              = errors.New("hệ thống đang bận")
)
