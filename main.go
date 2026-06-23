package main

import (
	"fmt"
	"image"
	"log"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type AudioStream struct {
	ID        int
	Name      string
	Direction string
}

func scanStreams(command string, direction string, idRegex *regexp.Regexp) ([]AudioStream, error) {
	out, err := exec.Command("pactl", "list", command).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run pactl list %s: %w", command, err)
	}

	var streams []AudioStream

	nameRegex := regexp.MustCompile(`application\.(?:name|process\.binary) = "([^"]+)"`)

	blocks := strings.Split(string(out), "\n\n")

	for _, block := range blocks {
		idMatch := idRegex.FindStringSubmatch(block)
		nameMatch := nameRegex.FindStringSubmatch(block)

		if idMatch != nil && nameMatch != nil {
			id, _ := strconv.Atoi(idMatch[1])
			streams = append(streams, AudioStream{
				ID:        id,
				Name:      nameMatch[1],
				Direction: direction,
			})
		}
	}
	return streams, nil
}

func setStreamVolume(ID int, volume int) {
	ID_S := strconv.Itoa(ID)
	VOLUME_S := strconv.Itoa(volume)
	VOLUME_S += "%"

	_, _ = exec.Command("pactl", "set-sink-input-volume", ID_S, VOLUME_S).Output()
}

func getStreamVolumeByID(id int) int {
	out, err := exec.Command("pactl", "list", "sink-inputs").Output()
	if err != nil {
		panic(err)
	}

	blocks := strings.Split(string(out), "\n\n")

	targetHeader := fmt.Sprintf("Sink Input #%d", id)

	volRegex := regexp.MustCompile(`Volume:.*?(\d+)%`)

	for _, block := range blocks {
		if strings.Contains(block, targetHeader) {
			if volMatch := volRegex.FindStringSubmatch(block); len(volMatch) > 1 {
				volume, err := strconv.Atoi(volMatch[1])
				if err != nil {
					panic(err)
				}

				return volume
			}
		}
	}

	return 0
}

type Game struct{}

func (g *Game) Update() error {
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {

}

func (g *Game) Layout(ow, oh int) (sw, sh int) {
	return 1, 1
}

type Root struct {
	guigui.DefaultWidget

	background basicwidget.Background

	sliders guigui.WidgetSlice[*basicwidget.Slider]
	text    guigui.WidgetSlice[*basicwidget.Text]

	Volumes map[int]Used_Index

	Rows        guigui.LinearLayout
	RowsItems   []guigui.LinearLayoutItem
	layoutItems []guigui.LinearLayoutItem

	SlectedChannel int

	game *Game
}

type Used_Index struct {
	Volume  int
	Name    string
	AudioID int
}

func (r *Root) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			r.SlectedChannel -= 1
		} else {
			r.SlectedChannel += 1
		}
		guigui.RequestRedraw(r)
		guigui.RequestRebuild(r)
	}

	if r.SlectedChannel < 0 {
		r.SlectedChannel = 0
		guigui.RequestRedraw(r)
		guigui.RequestRebuild(r)
	} else if r.SlectedChannel > r.sliders.Len()-1 {
		r.SlectedChannel = r.sliders.Len() - 1
		guigui.RequestRedraw(r)
		guigui.RequestRebuild(r)
	}

	for i := range r.sliders.Len() {
		if i == r.SlectedChannel {
			if ebiten.IsKeyPressed(ebiten.Key1) {
				if r.Volumes[i].Volume-2 >= 0 {
					setStreamVolume(r.Volumes[i].AudioID, r.Volumes[i].Volume-2)
					guigui.RequestRedraw(r)
					guigui.RequestRebuild(r)
				} else {
					setStreamVolume(r.Volumes[i].AudioID, 0)
					guigui.RequestRedraw(r)
					guigui.RequestRebuild(r)
				}
			}
			if ebiten.IsKeyPressed(ebiten.Key3) {
				if r.Volumes[i].Volume+2 <= 100 {
					setStreamVolume(r.Volumes[i].AudioID, r.Volumes[i].Volume+2)
					guigui.RequestRedraw(r)
					guigui.RequestRebuild(r)
				} else {
					setStreamVolume(r.Volumes[i].AudioID, 100)
					guigui.RequestRedraw(r)
					guigui.RequestRebuild(r)
				}
			}
		}
	}

	return nil
}

func (r *Root) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&r.background)

	if r.game == nil {
		r.game = &Game{}
	}

	idReSink := regexp.MustCompile(`Sink Input #(\d+)`)
	playbackStreams, err := scanStreams("sink-inputs", "Playback", idReSink)
	if err != nil {
		log.Printf("Warning: Could not scan playback: %v", err)
	}

	r.Volumes = make(map[int]Used_Index)

	r.sliders.SetLen(len(playbackStreams))
	r.text.SetLen(len(playbackStreams))
	for i := range r.sliders.Len() {
		adder.AddWidget(r.sliders.At(i))
		adder.AddWidget(r.text.At(i))
		r.Volumes[i] = Used_Index{getStreamVolumeByID(playbackStreams[i].ID), playbackStreams[i].Name, playbackStreams[i].ID}
	}

	for i := range len(playbackStreams) {
		volume := r.Volumes[i]

		slider_to_add := r.sliders.At(i)

		slider_to_add.OnValueChanged(func(context *guigui.Context, value int) {
			setStreamVolume(volume.AudioID, value)
		})

		slider_to_add.SetValue(volume.Volume)
		slider_to_add.SetMaximumValue(100)
		slider_to_add.SetMinimumValue(0)

		text_to_add := r.text.At(i)
		text_to_add.SetValue(volume.Name)
		text_to_add.SetScale(2)

		if i == r.SlectedChannel {
			fmt.Println("testing")
			text_to_add.SetBold(true)
		} else {
			text_to_add.SetBold(false)
		}
	}

	return nil
}

func (r *Root) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&r.background, widgetBounds.Bounds())

	unit_size := basicwidget.UnitSize(context)

	r.RowsItems = slices.Delete(r.RowsItems, 0, len(r.RowsItems))

	for i := range r.sliders.Len() {
		w := widgetBounds.Bounds().Dx()
		h := r.text.At(i).Measure(context, guigui.FixedWidthConstraints(w)).Y
		r.RowsItems = append(r.RowsItems, guigui.LinearLayoutItem{
			Widget: r.text.At(i),
			Size:   guigui.FixedSize(h),
		})
		w = widgetBounds.Bounds().Dx()
		h = r.sliders.At(i).Measure(context, guigui.FixedWidthConstraints(w)).Y
		r.RowsItems = append(r.RowsItems, guigui.LinearLayoutItem{
			Widget: r.sliders.At(i),
			Size:   guigui.FixedSize(h),
		})
	}

	r.Rows = guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items:     r.RowsItems,
		Gap:       unit_size / 2,
	}

	r.layoutItems = slices.Delete(r.layoutItems, 0, len(r.layoutItems))
	r.layoutItems = append(r.layoutItems,
		guigui.LinearLayoutItem{
			Size:   guigui.FixedSize(unit_size),
			Layout: &r.Rows,
		},
	)

	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     r.RowsItems,
		Gap:       unit_size,
		Padding: guigui.Padding{
			Start:  unit_size,
			Top:    unit_size,
			End:    unit_size,
			Bottom: unit_size,
		},
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func main() {
	op := &guigui.RunOptions{
		Title:          "pavu osd",
		WindowMinSize:  image.Pt(1280, 720),
		RunGameOptions: &ebiten.RunGameOptions{},
	}

	root := &Root{}

	root.SlectedChannel = 0

	if err := guigui.Run(root, op); err != nil {
		panic(err)
	}

}
