package trees

import (
	"github.com/fogleman/gg"
)

type TreeNode struct {
	Name     string
	Children []*TreeNode
}

type Member struct {
	Parent string
	Name   string
}

type OriginType struct {
	X float64
	Y float64
}

func drawBox(dc *gg.Context, row, boxesInRow, index int64, gapX, gapY, rectW, rectH, paddingTop, imgW, imgH, lineOriginX, lineOriginY float64, text string) (float64, float64) {
	var half float64 = float64(boxesInRow / 2)

	anchorX := imgW / 2
	gapsCount := float64(boxesInRow - 1)

	var x float64

	if boxesInRow%2 == 0 {
		if float64(index) <= half {
			x = anchorX - (rectW / 2) - ((half - float64(index)) * rectW) - (((gapsCount / 2) - float64(index-1)) * gapX) - (rectW / 2)
		} else {
			x = anchorX - (rectW / 2) - ((half - float64(index) + 1) * rectW) - (((gapsCount / 2) - float64(index-1)) * gapX) + (rectW / 2)
		}
	} else {
		x = anchorX - (rectW / 2) - ((half - float64(index) + 1) * rectW) - (((gapsCount / 2) - float64(index-1)) * gapX)
	}

	var y float64 = paddingTop + ((rectH + gapY) * float64(row-1))

	dc.DrawRoundedRectangle(x, y, rectW, rectH, ((rectW+rectH)/2)/12)
	// Background
	dc.SetRGB(1, 1, 1)
	dc.Fill()

	// Border
	dc.DrawRoundedRectangle(x, y, rectW, rectH, ((rectW+rectH)/2)/12)
	dc.SetRGB(0, 0, 0)
	dc.SetLineWidth(2)
	dc.Stroke()

	if row > 1 {
		lineStrokeWidth := 1.0
		dc.DrawLine(lineOriginX, lineOriginY+lineStrokeWidth, x+(rectW/2), y-lineStrokeWidth)
		dc.SetRGB(1, 1, 1)
		dc.SetLineWidth(lineStrokeWidth)
		dc.Stroke()
	}

	fontPaddingX := 5.0

	dc.SetRGB(0, 0, 0)
	dc.DrawStringWrapped(text, x+(rectW/2), y+(rectH/2), 0.5, 0.5, rectW-fontPaddingX, 1.0, gg.AlignCenter)

	return x + rectW/2, y + rectH
}

func traverseTree(t *TreeNode, parent string, a *[][]Member, i int) {
	if len(*a) < i+1 {
		*a = append(*a, []Member{
			{
				Name:   t.Name,
				Parent: parent,
			},
		})
	} else {
		(*a)[i] = append((*a)[i], Member{
			Name:   t.Name,
			Parent: parent,
		})
	}

	for _, child := range t.Children {
		traverseTree(child, t.Name, a, i+1)
	}
}

func maxRowElements(a *[][]Member) int {
	var max int

	for _, members := range *a {
		length := len(members)
		if length > max {
			max = length
		}
	}

	return max
}

func DrawTree(tree *TreeNode, rectH, rectW, gapX, gapY, paddingTop, paddingBottom, paddingLeft, paddingRight float64, outputName string) {
	var treeList [][]Member

	traverseTree(tree, "", &treeList, 0)

	rowCount := float64(len(treeList))
	maxRowElementsCount := float64(maxRowElements(&treeList))

	width := (maxRowElementsCount * rectW) + ((maxRowElementsCount - 1) * gapX) + paddingLeft + paddingRight
	height := (rowCount*rectH + (rowCount-1)*gapY) + paddingTop + paddingBottom

	dc := gg.NewContext(int(width), int(height))

	dc.SetHexColor("#36393f")
	dc.Clear()

	origins := make(map[string]OriginType)
	originX, originY := width/2, 0.0
	origins[""] = OriginType{X: originX, Y: originY}

	for index, members := range treeList {
		row := index + 1

		for memberIndex, member := range members {
			origin := origins[member.Parent]

			originX, originY = drawBox(dc, int64(row), int64(len(members)), int64(memberIndex+1), gapX, gapY, rectW, rectH, paddingTop, width, height, origin.X, origin.Y, member.Name)

			origins[member.Name] = OriginType{X: originX, Y: originY}
		}
	}

	dc.SavePNG(outputName)
}
