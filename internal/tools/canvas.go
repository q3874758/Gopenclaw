package tools

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
)

// CanvasExecutor Canvas 工具执行器
type CanvasExecutor struct {
	BaseExecutor
}

// NewCanvasExecutor 创建 Canvas 执行器
func NewCanvasExecutor() *CanvasExecutor {
	return &CanvasExecutor{
		BaseExecutor: BaseExecutor{
			name:        "canvas",
			description: "Draw, annotate, or manipulate images on a canvas",
		},
	}
}

// Execute 执行 Canvas 操作
func (e *CanvasExecutor) Execute(ctx context.Context, args map[string]any) (string, error) {
	action, _ := args["action"].(string)
	if action == "" {
		return "", fmt.Errorf("action required")
	}

	switch action {
	case "draw_rect":
		return e.drawRect(args)
	case "draw_circle":
		return e.drawCircle(args)
	case "draw_line":
		return e.drawLine(args)
	case "composite":
		return e.composite(args)
	case "load":
		return e.loadImage(args)
	case "save":
		return e.saveImage(args)
	default:
		return "", fmt.Errorf("unknown action: %s", action)
	}
}

// drawRect 绘制矩形
func (e *CanvasExecutor) drawRect(args map[string]any) (string, error) {
	imgData, _ := args["image"].(string)
	x, _ := args["x"].(float64)
	y, _ := args["y"].(float64)
	width, _ := args["width"].(float64)
	height, _ := args["height"].(float64)
	colorStr, _ := args["color"].(string)

	img, err := e.decodeImage(imgData)
	if err != nil {
		return "", err
	}

	rect := image.Rect(int(x), int(y), int(x+width), int(y+height))
	c := parseColor(colorStr)

	draw.Draw(img, rect, &image.Uniform{c}, image.Point{}, draw.Src)

	return e.encodeImage(img)
}

// drawCircle 绘制圆形
func (e *CanvasExecutor) drawCircle(args map[string]any) (string, error) {
	imgData, _ := args["image"].(string)
	cx, _ := args["cx"].(float64)
	cy, _ := args["cy"].(float64)
	radius, _ := args["radius"].(float64)
	colorStr, _ := args["color"].(string)

	img, err := e.decodeImage(imgData)
	if err != nil {
		return "", err
	}

	c := parseColor(colorStr)

	// 绘制实心圆
	for dy := -int(radius); dy <= int(radius); dy++ {
		for dx := -int(radius); dx <= int(radius); dx++ {
			if dx*dx+dy*dy <= int(radius*radius) {
				px := int(cx) + dx
				py := int(cy) + dy
				if px >= 0 && px < img.Bounds().Dx() && py >= 0 && py < img.Bounds().Dy() {
					img.Set(px, py, c)
				}
			}
		}
	}

	return e.encodeImage(img)
}

// drawLine 绘制线条
func (e *CanvasExecutor) drawLine(args map[string]any) (string, error) {
	imgData, _ := args["image"].(string)
	x1, _ := args["x1"].(float64)
	y1, _ := args["y1"].(float64)
	x2, _ := args["x2"].(float64)
	y2, _ := args["y2"].(float64)
	colorStr, _ := args["color"].(string)

	img, err := e.decodeImage(imgData)
	if err != nil {
		return "", err
	}

	c := parseColor(colorStr)

	// Bresenham 算法绘制直线
	dx := int(x2 - x1)
	dy := int(y2 - y1)
	steps := abs(dx)
	if abs(dy) > steps {
		steps = abs(dy)
	}

	xInc := dx / steps
	yInc := dy / steps

	x, y := int(x1), int(y1)
	for i := 0; i <= steps; i++ {
		if x >= 0 && x < img.Bounds().Dx() && y >= 0 && y < img.Bounds().Dy() {
			img.Set(x, y, c)
		}
		x += xInc
		y += yInc
	}

	return e.encodeImage(img)
}

// composite 合成图片
func (e *CanvasExecutor) composite(args map[string]any) (string, error) {
	bgData, _ := args["background"].(string)
	fgData, _ := args["foreground"].(string)
	x, _ := args["x"].(float64)
	y, _ := args["y"].(float64)

	bg, err := e.decodeImage(bgData)
	if err != nil {
		return "", err
	}

	fg, err := e.decodeImage(fgData)
	if err != nil {
		return "", err
	}

	// 将 fg 合成到 bg 上
	offset := image.Point{int(x), int(y)}
	bounds := fg.Bounds().Add(offset)

	draw.Draw(bg, bounds, fg, image.Point{}, draw.Over)

	return e.encodeImage(bg)
}

// loadImage 从文件加载图片
func (e *CanvasExecutor) loadImage(args map[string]any) (string, error) {
	path, _ := args["path"].(string)
	if path == "" {
		return "", fmt.Errorf("path required")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(data), nil
}

// saveImage 保存图片到文件
func (e *CanvasExecutor) saveImage(args map[string]any) (string, error) {
	path, _ := args["path"].(string)
	imgData, _ := args["image"].(string)

	if path == "" || imgData == "" {
		return "", fmt.Errorf("path and image required")
	}

	data, err := base64.StdEncoding.DecodeString(imgData)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", err
	}

	return fmt.Sprintf(`{"saved":true,"path":"%s"}`, path), nil
}

// decodeImage 解码 base64 图片
func (e *CanvasExecutor) decodeImage(data string) (draw.Image, error) {
	if data == "" {
		// 创建空白图片
		return image.NewRGBA(image.Rect(0, 0, 800, 600)), nil
	}

	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}

	img, err := png.Decode(bytes.NewReader(decoded))
	if err != nil {
		return nil, err
	}

	// 转换为 RGBA
	rgba := image.NewRGBA(img.Bounds())
	draw.Draw(rgba, rgba.Bounds(), img, image.Point{}, draw.Src)
	return rgba, nil
}

// encodeImage 编码图片为 base64
func (e *CanvasExecutor) encodeImage(img image.Image) (string, error) {
	buf := new(bytes.Buffer)
	err := png.Encode(buf, img)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// parseColor 解析颜色字符串
func parseColor(colorStr string) color.Color {
	switch colorStr {
	case "red":
		return color.RGBA{255, 0, 0, 255}
	case "green":
		return color.RGBA{0, 255, 0, 255}
	case "blue":
		return color.RGBA{0, 0, 255, 255}
	case "black":
		return color.RGBA{0, 0, 0, 255}
	case "white":
		return color.RGBA{255, 255, 255, 255}
	case "yellow":
		return color.RGBA{255, 255, 0, 255}
	default:
		// 尝试解析 hex 颜色
		if len(colorStr) == 7 && colorStr[0] == '#' {
			var r, g, b uint8
			fmt.Sscanf(colorStr[1:], "%02x%02x%02x", &r, &g, &b)
			return color.RGBA{r, g, b, 255}
		}
		return color.RGBA{0, 0, 0, 255}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// GetSchema 返回工具 schema
func (e *CanvasExecutor) GetSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "canvas",
			"description": "Draw, annotate, or manipulate images. Supports drawing rectangles, circles, lines, and compositing images.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"action": map[string]any{
						"type":        "string",
						"description": "Action: draw_rect, draw_circle, draw_line, composite, load, save",
					},
					"image": map[string]any{
						"type":        "string",
						"description": "Base64 encoded image",
					},
					"x":        map[string]any{"type": "number"},
					"y":        map[string]any{"type": "number"},
					"width":    map[string]any{"type": "number"},
					"height":   map[string]any{"type": "number"},
					"color":    map[string]any{"type": "string"},
					"cx":       map[string]any{"type": "number"},
					"cy":       map[string]any{"type": "number"},
					"radius":   map[string]any{"type": "number"},
					"x1":       map[string]any{"type": "number"},
					"y1":       map[string]any{"type": "number"},
					"x2":       map[string]any{"type": "number"},
					"y2":       map[string]any{"type": "number"},
					"path":     map[string]any{"type": "string"},
					"background": map[string]any{"type": "string"},
					"foreground": map[string]any{"type": "string"},
				},
			},
		},
	}
}
