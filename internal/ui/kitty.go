package ui

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/charmbracelet/x/ansi"
	"github.com/sokinpui/coder/internal/project"
	"github.com/sokinpui/coder/internal/types"
)

const (
	placeholderRune  = "\U0010EEEE"
	maxPreviewPixels = 800
	maxPreviewBytes  = 512 * 1024
)

var kittyDiacritics = []rune{
	0x0305, 0x030D, 0x030E, 0x0310, 0x0312, 0x033D, 0x033E, 0x033F,
	0x0346, 0x034A, 0x034B, 0x034C, 0x0350, 0x0351, 0x0352, 0x0357,
	0x035B, 0x0363, 0x0364, 0x0365, 0x0366, 0x0367, 0x0368, 0x0369,
	0x036A, 0x036B, 0x036C, 0x036D, 0x036E, 0x036F, 0x0483, 0x0484,
	0x0485, 0x0486, 0x0487, 0x0592, 0x0593, 0x0594, 0x0595, 0x0597,
	0x0598, 0x0599, 0x059C, 0x059D, 0x059E, 0x05A0, 0x05A1, 0x05A8,
	0x05A9, 0x05AB, 0x05AC, 0x05AF, 0x05C4, 0x0610, 0x0611, 0x0612,
	0x0613, 0x0614, 0x0615, 0x0616, 0x0617, 0x0657, 0x0658, 0x0659,
	0x065A, 0x065B, 0x065D, 0x065E, 0x06D6, 0x06D7, 0x06D8, 0x06D9,
	0x06DA, 0x06DB, 0x06DC, 0x06DF, 0x06E0, 0x06E1, 0x06E2, 0x06E4,
	0x06E7, 0x06E8, 0x06EB, 0x06EC, 0x0730, 0x0732, 0x0733, 0x0735,
	0x0736, 0x073A, 0x073C, 0x073D, 0x073F, 0x0740, 0x0741, 0x0743,
	0x0745, 0x0747, 0x0749, 0x074A, 0x0840, 0x0841, 0x0842, 0x0843,
	0x0844, 0x0845, 0x0846, 0x0847, 0x0848, 0x0849, 0x084A, 0x084B,
	0x084C, 0x084D, 0x084E, 0x084F, 0x0850, 0x0851, 0x0852, 0x0853,
	0x0854, 0x0855, 0x08E4, 0x08E5, 0x08E6, 0x08E7, 0x08E8, 0x08E9,
	0x08EA, 0x08EB, 0x08EC, 0x08ED, 0x08EE, 0x08EF, 0x08F0, 0x08F1,
	0x08F2, 0x08F3, 0x08F4, 0x08F5, 0x08F6, 0x08F7, 0x08F8, 0x08F9,
	0x08FA, 0x08FB, 0x08FC, 0x08FD, 0x08FE, 0x08FF, 0x0951, 0x0953,
	0x0954, 0x0A02, 0x0A4C, 0x0A51, 0x0A70, 0x0A71, 0x0B01, 0x0B3F,
	0x0B44, 0x0B56, 0x0BAE, 0x0BF0, 0x0C00, 0x0C04, 0x0C3E, 0x0C40,
	0x0C46, 0x0C4A, 0x0C55, 0x0C56, 0x0D00, 0x0D02, 0x0D3B, 0x0D3C,
	0x0D41, 0x0D42, 0x0D43, 0x0D44, 0x0D4D, 0x0D62, 0x0D63, 0x0E31,
	0x0E34, 0x0E35, 0x0E36, 0x0E37, 0x0E4D, 0x0EB1, 0x0EB4, 0x0EB5,
	0x0EB6, 0x0EB7, 0x0EBB, 0x0EBC, 0x0ECD, 0x0F18, 0x0F19, 0x0F35,
	0x0F37, 0x0F39, 0x0F71, 0x0F72, 0x0F74, 0x0F7A, 0x0F7B, 0x0F7C,
	0x0F7D, 0x0F80, 0x0F82, 0x0F83, 0x0F84, 0x0F86, 0x0F87, 0x0FC6,
	0x1037, 0x1039, 0x103A, 0x108D, 0x135D, 0x135E, 0x135F, 0x1714,
	0x1734, 0x17D2, 0x19B5, 0x19B7, 0x19BA, 0x19C4, 0x19C5, 0x1A60,
	0x1A75, 0x1A76, 0x1A77, 0x1A78, 0x1A79, 0x1A7A, 0x1A7B, 0x1A7C,
	0x1B6B, 0x1B6D, 0x1B6E, 0x1B6F, 0x1B70, 0x1B71, 0x1B72, 0x1B73,
	0x1CD0, 0x1CD1, 0x1CD2, 0x1CD4, 0x1CD5, 0x1CD6, 0x1CD7, 0x1CD8,
	0x1CD9, 0x1CDA, 0x1CDB, 0x1CDC, 0x1CDD, 0x1CDE, 0x1CDF, 0x1CE0,
	0x1CE2, 0x1CE3, 0x1CE4, 0x1CE5, 0x1CE6, 0x1CE7, 0x1CE8, 0x1CED,
	0x1CF4, 0x1CF8, 0x1CF9, 0x1DC4, 0x1DC5, 0x1DC6, 0x1DC7, 0x1DC8,
	0x1DC9, 0x1DCB, 0x1DCC, 0x1DD1, 0x1DD2, 0x1DD3, 0x1DD4, 0x1DD5,
	0x1DD6, 0x1DD7, 0x1DD8, 0x1DD9, 0x1DDA, 0x1DDB, 0x1DDC, 0x1DDD,
	0x1DDE, 0x1DDF, 0x1DE0, 0x1DE1, 0x1DE2, 0x1DE3, 0x1DE4, 0x1DE5,
	0x1DE6, 0x1DFE, 0x20D0, 0x20D1, 0x20D4, 0x20D5, 0x20D6, 0x20D7,
	0x20DB, 0x20DC, 0x20E1, 0x20E7, 0x20E9, 0x20F0, 0x2CEF, 0x2CF0,
	0x2CF1, 0x2DE0, 0x2DE1, 0x2DE2, 0x2DE3, 0x2DE4, 0x2DE5, 0x2DE6,
	0x2DE7, 0x2DE8, 0x2DE9, 0x2DEA, 0x2DEB, 0x2DEC, 0x2DED, 0x2DEE,
	0x2DEF, 0x2DF0, 0x2DF1, 0x2DF2, 0x2DF3, 0x2DF4, 0x2DF5, 0x2DF6,
	0x2DF7, 0x2DF8, 0x2DF9, 0x2DFA, 0x2DFB, 0x2DFC, 0x2DFD, 0x2DFE,
	0x2DFF, 0xA66F, 0xA674, 0xA675, 0xA676, 0xA677, 0xA678, 0xA679,
	0xA67A, 0xA67B, 0xA67C, 0xA67D, 0xA69E, 0xA69F, 0xA6F0, 0xA6F1,
	0xFE20, 0xFE21, 0xFE22, 0xFE23, 0xFE24, 0xFE25, 0xFE26,
}

func diacritic(n int) rune {
	if n < 0 || n >= len(kittyDiacritics) {
		return kittyDiacritics[0]
	}
	return kittyDiacritics[n]
}

type cachedImage struct {
	cacheKey string
	pngData  []byte
	width    int
	height   int
	imageID  int
	failed   bool
}

var (
	imgCacheMu         sync.RWMutex
	imgCache           = make(map[string]*cachedImage)
	placementMu        sync.Mutex
	uploadedPlacements = make(map[int]string)
)

func ImageMessageCacheKey(msg types.Message) string {
	if msg.Content != "" {
		return msg.Content
	}
	if len(msg.Data) == 0 {
		return ""
	}
	h := sha256.Sum256(msg.Data)
	return hex.EncodeToString(h[:])
}

func PDFMessageCacheKey(msg types.Message) string {
	meta, ok := types.ParsePDFMetadata(msg.Content)
	if !ok || len(meta.Files) == 0 {
		return ""
	}
	firstPage := meta.Files[0]
	if !filepath.IsAbs(firstPage) {
		if root := project.Root(); root != "" {
			firstPage = filepath.Join(root, firstPage)
		}
	}
	return firstPage
}

func IsKittyImageCached(cacheKey string) bool {
	if cacheKey == "" {
		return false
	}
	imgCacheMu.RLock()
	_, ok := imgCache[cacheKey]
	imgCacheMu.RUnlock()
	return ok
}

func ProcessKittyImageMessage(msg types.Message) bool {
	cacheKey := ImageMessageCacheKey(msg)
	if cacheKey == "" {
		return false
	}
	if IsKittyImageCached(cacheKey) {
		return true
	}
	data, err := getImageBytes(msg)
	if err != nil || len(data) == 0 {
		imgCacheMu.Lock()
		imgCache[cacheKey] = &cachedImage{cacheKey: cacheKey, failed: true}
		imgCacheMu.Unlock()
		return false
	}
	if _, err := getOrProcessImage(data, cacheKey); err != nil {
		imgCacheMu.Lock()
		imgCache[cacheKey] = &cachedImage{cacheKey: cacheKey, failed: true}
		imgCacheMu.Unlock()
		return false
	}
	return true
}

func ProcessKittyImageFile(filePath string) bool {
	if filePath == "" {
		return false
	}
	if IsKittyImageCached(filePath) {
		return true
	}
	data, err := os.ReadFile(filePath)
	if err != nil || len(data) == 0 {
		imgCacheMu.Lock()
		imgCache[filePath] = &cachedImage{cacheKey: filePath, failed: true}
		imgCacheMu.Unlock()
		return false
	}
	if _, err := getOrProcessImage(data, filePath); err != nil {
		imgCacheMu.Lock()
		imgCache[filePath] = &cachedImage{cacheKey: filePath, failed: true}
		imgCacheMu.Unlock()
		return false
	}
	return true
}

func RenderKittyImageMessageCached(msg types.Message, viewportWidth int) (string, bool) {
	cacheKey := ImageMessageCacheKey(msg)
	if cacheKey == "" {
		return "", false
	}
	imgCacheMu.RLock()
	cached, ok := imgCache[cacheKey]
	imgCacheMu.RUnlock()
	if !ok || cached.failed {
		return "", false
	}
	title := msg.Content
	if title == "" {
		title = "Image"
	}
	return renderFromCachedImage(cached, title, viewportWidth, false)
}

func RenderKittyPDFCached(msg types.Message, viewportWidth int) (string, bool) {
	cacheKey := PDFMessageCacheKey(msg)
	if cacheKey == "" {
		return "", false
	}
	imgCacheMu.RLock()
	cached, ok := imgCache[cacheKey]
	imgCacheMu.RUnlock()
	if !ok || cached.failed {
		return "", false
	}
	meta, ok := types.ParsePDFMetadata(msg.Content)
	if !ok {
		return "", false
	}
	return renderFromCachedImage(cached, formatPDFHeader(meta), viewportWidth, false)
}

func EnsureKittyImageUploaded(cacheKey string, viewportWidth int) {
	if cacheKey == "" {
		return
	}
	imgCacheMu.RLock()
	img, ok := imgCache[cacheKey]
	imgCacheMu.RUnlock()
	if !ok || img.failed || len(img.pngData) == 0 {
		return
	}
	cols, rows := kittyCellDimensions(img, viewportWidth)
	placementKey := fmt.Sprintf("%s:%d:%d", img.cacheKey, cols, rows)

	placementMu.Lock()
	if uploadedPlacements[img.imageID] != placementKey {
		transmitKittyPlacement(img.imageID, img.pngData, cols, rows)
		uploadedPlacements[img.imageID] = placementKey
	}
	placementMu.Unlock()
}

func RenderKittyImageMessage(msg types.Message, viewportWidth int) (string, bool) {
	cacheKey := ImageMessageCacheKey(msg)
	if cacheKey != "" {
		imgCacheMu.RLock()
		cached, ok := imgCache[cacheKey]
		imgCacheMu.RUnlock()
		if ok {
			if cached.failed {
				return "", false
			}
			title := msg.Content
			if title == "" {
				title = "Image"
			}
			return renderFromCachedImage(cached, title, viewportWidth, true)
		}
	}

	data, err := getImageBytes(msg)
	if err != nil || len(data) == 0 {
		return "", false
	}
	title := msg.Content
	if title == "" {
		title = "Image"
	}
	return RenderKittyImageData(data, title, viewportWidth, cacheKey)
}

func RenderKittyImageFile(filePath, title string, viewportWidth int) (string, bool) {
	imgCacheMu.RLock()
	cached, ok := imgCache[filePath]
	imgCacheMu.RUnlock()
	if ok {
		if cached.failed {
			return "", false
		}
		return renderFromCachedImage(cached, title, viewportWidth, true)
	}

	data, err := os.ReadFile(filePath)
	if err != nil || len(data) == 0 {
		return "", false
	}
	return RenderKittyImageData(data, title, viewportWidth, filePath)
}

func PreprocessKittyImage(data []byte, cacheKey string, viewportWidth int) {
	if len(data) == 0 {
		return
	}
	img, err := getOrProcessImage(data, cacheKey)
	if err != nil {
		return
	}

	cols, rows := kittyCellDimensions(img, viewportWidth)
	placementKey := fmt.Sprintf("%s:%d:%d", img.cacheKey, cols, rows)

	placementMu.Lock()
	if uploadedPlacements[img.imageID] != placementKey {
		transmitKittyPlacement(img.imageID, img.pngData, cols, rows)
		uploadedPlacements[img.imageID] = placementKey
	}
	placementMu.Unlock()
}

func RenderKittyImageData(data []byte, title string, viewportWidth int, cacheKey ...string) (string, bool) {
	img, err := getOrProcessImage(data, cacheKey...)
	if err != nil {
		return "", false
	}
	return renderFromCachedImage(img, title, viewportWidth, true)
}

func getOrProcessImage(data []byte, cacheKey ...string) (*cachedImage, error) {
	key := ""
	if len(cacheKey) > 0 && cacheKey[0] != "" {
		key = cacheKey[0]
	} else {
		h := sha256.Sum256(data)
		key = hex.EncodeToString(h[:])
	}

	imgCacheMu.RLock()
	if cached, ok := imgCache[key]; ok && !cached.failed {
		imgCacheMu.RUnlock()
		return cached, nil
	}
	imgCacheMu.RUnlock()

	pngData, width, height, err := ensurePNG(data)
	if err != nil {
		return nil, err
	}

	img := &cachedImage{
		cacheKey: key,
		pngData:  pngData,
		width:    width,
		height:   height,
		imageID:  computeImageID(pngData),
	}

	imgCacheMu.Lock()
	imgCache[key] = img
	imgCacheMu.Unlock()
	return img, nil
}

func kittyCellDimensions(img *cachedImage, viewportWidth int) (int, int) {
	width := 80
	if viewportWidth > 0 {
		width = viewportWidth
	}
	cardWidth := max(24, min(width-4, 80))
	contentWidth := max(10, cardWidth-ImageMessageStyle.GetHorizontalFrameSize())
	maxCols := min(contentWidth, 50)
	maxRows := 18
	return calculateCellDimensions(img.width, img.height, maxCols, maxRows)
}

func renderFromCachedImage(img *cachedImage, title string, viewportWidth int, transmit bool) (string, bool) {
	cardWidth := max(24, min(viewportWidth-4, 80))
	contentWidth := max(10, cardWidth-ImageMessageStyle.GetHorizontalFrameSize())
	cols, rows := kittyCellDimensions(img, viewportWidth)
	if transmit {
		placementKey := fmt.Sprintf("%s:%d:%d", img.cacheKey, cols, rows)
		placementMu.Lock()
		if uploadedPlacements[img.imageID] != placementKey {
			transmitKittyPlacement(img.imageID, img.pngData, cols, rows)
			uploadedPlacements[img.imageID] = placementKey
		}
		placementMu.Unlock()
	}

	headerText := fmt.Sprintf("%s (%dx%d)", project.ShortenPath(title), img.width, img.height)
	if ansi.StringWidth(headerText) > contentWidth {
		headerText = ansi.Truncate(headerText, contentWidth, "…")
	}

	var sb strings.Builder
	sb.WriteString(ToolMutedStyle.Render(headerText))
	for r := range rows {
		sb.WriteByte('\n')
		sb.WriteString(renderPlaceholderRow(img.imageID, r, cols))
	}

	return ImageMessageStyle.Width(cardWidth).Render(sb.String()), true
}

func transmitKittyPlacement(imageID int, pngData []byte, cols, rows int) {
	payload := base64.StdEncoding.EncodeToString(pngData)
	const chunkSize = 4096
	total := len(payload)

	for offset := 0; offset < total; offset += chunkSize {
		end := min(offset+chunkSize, total)
		more := 1
		if end >= total {
			more = 0
		}
		chunk := payload[offset:end]

		var seq string
		if offset == 0 {
			seq = fmt.Sprintf("\x1b_Ga=T,f=100,t=d,i=%d,U=1,c=%d,r=%d,q=2,m=%d;%s\x1b\\", imageID, cols, rows, more, chunk)
		} else {
			seq = fmt.Sprintf("\x1b_Gm=%d,q=2;%s\x1b\\", more, chunk)
		}
		_, _ = os.Stderr.WriteString(wrapTmux(seq))
	}
}

func renderPlaceholderRow(imageID, r, cols int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\x1b[38;5;%dm", imageID))
	for c := range cols {
		sb.WriteString(placeholderRune)
		sb.WriteRune(diacritic(r))
		sb.WriteRune(diacritic(c))
	}
	sb.WriteString("\x1b[39m")
	return sb.String()
}

func computeImageID(data []byte) int {
	var h uint32 = 2166136261
	for _, b := range data {
		h ^= uint32(b)
		h *= 16777619
	}
	return int((h % 240) + 1)
}

func calculateCellDimensions(imgWidth, imgHeight, maxCols, maxRows int) (int, int) {
	if imgWidth <= 0 || imgHeight <= 0 {
		return 30, 10
	}
	if maxCols <= 0 {
		maxCols = 40
	}
	if maxRows <= 0 {
		maxRows = 20
	}

	const cellAspect = 0.5
	visualRatio := (float64(imgHeight) / float64(imgWidth)) * cellAspect

	cols := maxCols
	rows := int(float64(cols) * visualRatio)

	if rows > maxRows {
		rows = maxRows
		cols = int(float64(rows) / visualRatio)
	}

	if cols > maxCols {
		cols = maxCols
	}

	if cols < 4 {
		cols = 4
	}
	if rows < 1 {
		rows = 1
	}

	return cols, rows
}

func ensurePNG(data []byte) ([]byte, int, int, error) {
	if len(data) == 0 {
		return nil, 0, 0, fmt.Errorf("empty image data")
	}

	if bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) {
		cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
		if err == nil && cfg.Width > 0 && cfg.Height > 0 && cfg.Width <= maxPreviewPixels && cfg.Height <= maxPreviewPixels && len(data) <= maxPreviewBytes {
			return data, cfg.Width, cfg.Height, nil
		}
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, err
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return nil, 0, 0, fmt.Errorf("invalid image dimensions: %dx%d", width, height)
	}

	previewImg := img
	if width > maxPreviewPixels || height > maxPreviewPixels {
		previewImg = downscaleImage(img, maxPreviewPixels)
	}

	var buf bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := encoder.Encode(&buf, previewImg); err != nil {
		return nil, 0, 0, err
	}

	return buf.Bytes(), width, height, nil
}

func downscaleImage(src image.Image, maxDim int) image.Image {
	bounds := src.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()
	if srcW <= maxDim && srcH <= maxDim {
		return src
	}

	dstW, dstH := srcW, srcH
	if srcW >= srcH {
		dstW = maxDim
		dstH = max(1, (srcH*maxDim)/srcW)
	} else {
		dstH = maxDim
		dstW = max(1, (srcW*maxDim)/srcH)
	}

	dst := image.NewNRGBA(image.Rect(0, 0, dstW, dstH))
	switch s := src.(type) {
	case *image.NRGBA:
		downscaleNRGBA(dst, s, bounds, srcW, srcH, dstW, dstH)
		return dst
	case *image.RGBA:
		downscaleRGBA(dst, s, bounds, srcW, srcH, dstW, dstH)
		return dst
	case *image.YCbCr:
		downscaleYCbCr(dst, s, bounds, srcW, srcH, dstW, dstH)
		return dst
	}

	for dy := range dstH {
		sy0 := bounds.Min.Y + (dy*srcH)/dstH
		sy1 := max(sy0+1, bounds.Min.Y+((dy+1)*srcH)/dstH)
		sy1 = min(sy1, bounds.Max.Y)

		for dx := range dstW {
			sx0 := bounds.Min.X + (dx*srcW)/dstW
			sx1 := max(sx0+1, bounds.Min.X+((dx+1)*srcW)/dstW)
			sx1 = min(sx1, bounds.Max.X)

			var rSum, gSum, bSum, aSum, count uint64
			for sy := sy0; sy < sy1; sy++ {
				for sx := sx0; sx < sx1; sx++ {
					r, g, b, a := readNRGBAPixel(src.At(sx, sy))
					rSum += uint64(r)
					gSum += uint64(g)
					bSum += uint64(b)
					aSum += uint64(a)
					count++
				}
			}

			if count == 0 {
				continue
			}
			dst.SetNRGBA(dx, dy, color.NRGBA{
				R: uint8(rSum / count),
				G: uint8(gSum / count),
				B: uint8(bSum / count),
				A: uint8(aSum / count),
			})
		}
	}

	return dst
}

func downscaleNRGBA(dst, src *image.NRGBA, bounds image.Rectangle, srcW, srcH, dstW, dstH int) {
	for dy := range dstH {
		sy0 := bounds.Min.Y + (dy*srcH)/dstH
		sy1 := min(bounds.Max.Y, max(sy0+1, bounds.Min.Y+((dy+1)*srcH)/dstH))
		dstRowOff := dy * dst.Stride

		for dx := range dstW {
			sx0 := bounds.Min.X + (dx*srcW)/dstW
			sx1 := min(bounds.Max.X, max(sx0+1, bounds.Min.X+((dx+1)*srcW)/dstW))

			var rSum, gSum, bSum, aSum, count uint64
			for sy := sy0; sy < sy1; sy++ {
				rowStart := (sy-src.Rect.Min.Y)*src.Stride + (sx0-src.Rect.Min.X)*4
				for sx := sx0; sx < sx1; sx++ {
					rSum += uint64(src.Pix[rowStart])
					gSum += uint64(src.Pix[rowStart+1])
					bSum += uint64(src.Pix[rowStart+2])
					aSum += uint64(src.Pix[rowStart+3])
					rowStart += 4
					count++
				}
			}
			if count == 0 {
				continue
			}
			dstOff := dstRowOff + dx*4
			dst.Pix[dstOff] = uint8(rSum / count)
			dst.Pix[dstOff+1] = uint8(gSum / count)
			dst.Pix[dstOff+2] = uint8(bSum / count)
			dst.Pix[dstOff+3] = uint8(aSum / count)
		}
	}
}

func downscaleRGBA(dst *image.NRGBA, src *image.RGBA, bounds image.Rectangle, srcW, srcH, dstW, dstH int) {
	for dy := range dstH {
		sy0 := bounds.Min.Y + (dy*srcH)/dstH
		sy1 := min(bounds.Max.Y, max(sy0+1, bounds.Min.Y+((dy+1)*srcH)/dstH))
		dstRowOff := dy * dst.Stride

		for dx := range dstW {
			sx0 := bounds.Min.X + (dx*srcW)/dstW
			sx1 := min(bounds.Max.X, max(sx0+1, bounds.Min.X+((dx+1)*srcW)/dstW))

			var rSum, gSum, bSum, aSum, count uint64
			for sy := sy0; sy < sy1; sy++ {
				rowStart := (sy-src.Rect.Min.Y)*src.Stride + (sx0-src.Rect.Min.X)*4
				for sx := sx0; sx < sx1; sx++ {
					r := uint64(src.Pix[rowStart])
					g := uint64(src.Pix[rowStart+1])
					b := uint64(src.Pix[rowStart+2])
					a := uint64(src.Pix[rowStart+3])
					if a > 0 && a < 255 {
						r = (r * 255) / a
						g = (g * 255) / a
						b = (b * 255) / a
					}
					rSum += r
					gSum += g
					bSum += b
					aSum += a
					rowStart += 4
					count++
				}
			}
			if count == 0 {
				continue
			}
			dstOff := dstRowOff + dx*4
			dst.Pix[dstOff] = uint8(rSum / count)
			dst.Pix[dstOff+1] = uint8(gSum / count)
			dst.Pix[dstOff+2] = uint8(bSum / count)
			dst.Pix[dstOff+3] = uint8(aSum / count)
		}
	}
}

func downscaleYCbCr(dst *image.NRGBA, src *image.YCbCr, bounds image.Rectangle, srcW, srcH, dstW, dstH int) {
	for dy := range dstH {
		sy0 := bounds.Min.Y + (dy*srcH)/dstH
		sy1 := min(bounds.Max.Y, max(sy0+1, bounds.Min.Y+((dy+1)*srcH)/dstH))
		dstRowOff := dy * dst.Stride

		for dx := range dstW {
			sx0 := bounds.Min.X + (dx*srcW)/dstW
			sx1 := min(bounds.Max.X, max(sx0+1, bounds.Min.X+((dx+1)*srcW)/dstW))

			var rSum, gSum, bSum, count uint64
			for sy := sy0; sy < sy1; sy++ {
				for sx := sx0; sx < sx1; sx++ {
					yi := src.YOffset(sx, sy)
					ci := src.COffset(sx, sy)
					r, g, b := color.YCbCrToRGB(src.Y[yi], src.Cb[ci], src.Cr[ci])
					rSum += uint64(r)
					gSum += uint64(g)
					bSum += uint64(b)
					count++
				}
			}
			if count == 0 {
				continue
			}
			dstOff := dstRowOff + dx*4
			dst.Pix[dstOff] = uint8(rSum / count)
			dst.Pix[dstOff+1] = uint8(gSum / count)
			dst.Pix[dstOff+2] = uint8(bSum / count)
			dst.Pix[dstOff+3] = 255
		}
	}
}

func readNRGBAPixel(c color.Color) (uint8, uint8, uint8, uint8) {
	if nrgba, ok := c.(color.NRGBA); ok {
		return nrgba.R, nrgba.G, nrgba.B, nrgba.A
	}
	nrgba := color.NRGBAModel.Convert(c).(color.NRGBA)
	return nrgba.R, nrgba.G, nrgba.B, nrgba.A
}

func getImageBytes(msg types.Message) ([]byte, error) {
	if len(msg.Data) > 0 {
		return msg.Data, nil
	}
	if msg.Content == "" {
		return nil, fmt.Errorf("empty image content and data")
	}

	path := msg.Content
	if !filepath.IsAbs(path) {
		if root := project.Root(); root != "" {
			path = filepath.Join(root, path)
		}
	}
	data, err := os.ReadFile(path)
	if err == nil && len(data) > 0 {
		return data, nil
	}

	if cwd, err := os.Getwd(); err == nil {
		if data, err := os.ReadFile(filepath.Join(cwd, msg.Content)); err == nil && len(data) > 0 {
			return data, nil
		}
	}
	return nil, fmt.Errorf("unable to read image file: %s", msg.Content)
}

func wrapTmux(seq string) string {
	if os.Getenv("TMUX") == "" {
		return seq
	}
	escaped := strings.ReplaceAll(seq, "\x1b", "\x1b\x1b")
	return "\x1bPtmux;" + escaped + "\x1b\\"
}

func ClearKittyGraphics() {
	placementMu.Lock()
	uploadedPlacements = make(map[int]string)
	placementMu.Unlock()

	seq := wrapTmux("\x1b_Ga=d,d=A\x1b\\")
	_, _ = os.Stdout.WriteString(seq)
}
