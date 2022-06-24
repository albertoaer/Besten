package lexer

import (
	"errors"
	"io"
)

type Source interface {
	Origin() string
	GetSource() (io.ReadCloser, error)
}

type Lexer struct {
	source Source
}

func LexerFor(source Source) *Lexer {
	return &Lexer{source: source}
}

type Block struct {
	Begin    int
	End      int
	Tokens   []Token
	Children []Block
	Parent   *Block
	Origin   string
}

func getChilds(blks []dirtyBlock) (line []Token, sub bool, children []dirtyBlock, lline int, err error) {
	sub = false
	for i := range blks {
		//Read and append the token to the main line
		tk, s, e := GetTokens(blks[i].raw)
		lline = blks[i].line
		if e != nil {
			err = e
			return
		}
		line = append(line, tk...)

		//Check subs or children
		if sub || len(blks[i].children) > 0 {
			if i == len(blks)-1 {
				sub = s
				children = blks[i].children
			} else {
				if sub {
					err = errors.New("Unexpected block opening")
				} else { //len(raw.children[i].children) > 0
					err = errors.New("Unexpected indentation")
				}
				return
			}
		} else if i == len(blks)-1 {
			children = make([]dirtyBlock, 0)
		}
	}
	return
}

func (l *Lexer) solveBlock(raw dirtyBlock, parent *Block) (block Block, err error) {
	//sublevel: Indicates that any subline is in a sublevel, otherwise sublines would be treat like the same line
	tks, sublevel, err := GetTokens(raw.raw)
	begin := raw.line
	end := raw.line
	if err != nil {
		return
	}

	//Multiline and pick children
	target_children := raw.children
	if !sublevel && len(raw.children) > 0 {
		l, sub, target, nline, e := getChilds(target_children)
		if nline > end {
			end = nline
		}
		if e != nil {
			err = e
			return
		}
		target_children = target
		sublevel = sub
		tks = append(tks, l...)
	}
	if sublevel != (len(target_children) > 0) {
		err = errors.New("No correspondence between indentation and block opening")
		return
	}

	//Solve children and fill the block template
	childs, err := l.solveBlocks(target_children, &block)
	if err != nil {
		return
	}
	block.Begin = begin
	block.End = end
	block.Parent = parent
	block.Tokens = tks
	block.Children = childs
	block.Origin = l.source.Origin()
	return
}

func (l *Lexer) solveBlocks(raw []dirtyBlock, parent *Block) (blocks []Block, err error) {
	for _, r := range raw {
		b, e := l.solveBlock(r, parent)
		if e != nil {
			err = e
			return
		}
		if len(b.Tokens) > 0 {
			blocks = append(blocks, b)
		}
	}
	return
}

func (l *Lexer) GetBlocks() (blocks []Block, err error) {
	var s io.ReadCloser
	if s, err = l.source.GetSource(); err != nil {
		return
	}
	raw_blocks, err := getRawStructure(s)
	if err != nil {
		return
	}
	blocks, err = l.solveBlocks(raw_blocks, nil)
	e := s.Close()
	if err != nil {
		err = e
	}
	return
}
