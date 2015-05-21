package main

import (
	"azul3d.org/gfx.v1"
	"azul3d.org/gfx/window.v2"
	"azul3d.org/keyboard.v1"
	"azul3d.org/lmath.v1"
	"azul3d.org/native/freetype.v1"
	"image"
	"io/ioutil"
	"log"
	"os"
)

var glslVert = []byte(`
#version 120

attribute vec3 Vertex;
attribute vec2 TexCoord0;
uniform mat4 MVP;
varying vec2 tc0;

void main() {
  tc0 = TexCoord0;
  gl_Position = MVP * vec4(Vertex, 1.0);
}
`)

var glslFrag = []byte(`
#version 120

varying vec2 tc0;
uniform sampler2D Texture0;
uniform bool BinaryAlpha;

void main() {
  gl_FragColor = texture2D(Texture0, tc0);
  if(BinaryAlpha && gl_FragColor.a < 0.5) {
		discard;
	}
}
`)

var (
	fontSize = 50 * 64
	font     *freetype.Font
	shader   *gfx.Shader
)

//
type Text struct {
	Object *gfx.Object
	Runes  []rune
}

func NewText() *Text {
	t := new(Text)
	t.Object = gfx.NewObject()
	t.Runes = make([]rune, 0)
	return t
}

//
func init() {
	file, err := os.Open("arial.ttf") // Please change here respectively
	if err != nil {
		log.Fatal(err)
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}
	c, err := freetype.Init()
	if err != nil {
		log.Fatal(err)
	}
	font, err = c.Load(data)
	if err != nil {
		log.Fatal(err)
	}

	shader = gfx.NewShader("SimpleShader")
	shader.GLSLVert = glslVert
	shader.GLSLFrag = glslFrag
}

//
func (t *Text) Update() error {
	meshes := make([]*gfx.Mesh, 0)
	textures := make([]*gfx.Texture, 0)
	dx := float32(0)

	for _, r := range t.Runes {
		font.SetSize(fontSize, 0, 72, 0)
		glyph, err := font.Load(font.Index(r))
		if err != nil {
			log.Fatal(err)
		}
		m, err := glyph.Image()
		if err != nil {
			log.Fatal(err)
		}
		tex := gfx.NewTexture()
		tex.Source = m
		tex.MinFilter = gfx.Nearest
		tex.Format = gfx.DXT1RGBA
		textures = append(textures, tex)

		hx := float32(glyph.Width / 128)
		hy := float32(glyph.Height / 128)
		dx += hx * float32(2)

		mesh := gfx.NewMesh()
		mesh.Vertices = []gfx.Vec3{
			{-hx + dx, 0, -hy},
			{hx + dx, 0, -hy},
			{-hx + dx, 0, hy},

			{-hx + dx, 0, hy},
			{hx + dx, 0, -hy},
			{hx + dx, 0, hy},
		}
		mesh.TexCoords = []gfx.TexCoordSet{{
			Slice: []gfx.TexCoord{
				{0, 1},
				{1, 1},
				{0, 0},

				{0, 0},
				{1, 1},
				{1, 0},
			},
		}}
		meshes = append(meshes, mesh)
	}

	obj := gfx.NewObject()
	obj.AlphaMode = gfx.BinaryAlpha

	obj.Shader = shader
	obj.Meshes = meshes
	obj.Textures = textures
	t.Object = obj
	return nil
}

//
func gfxLoop(w window.Window, r gfx.Renderer) {

	// Set up
	camera := gfx.NewCamera()
	camNear := 0.01
	camFar := 1000.0
	camera.SetOrtho(r.Bounds(), camNear, camFar)
	camera.SetPos(lmath.Vec3{0, -10, 0})

	text := NewText()

	// Notification
	e := make(chan interface{})
	go func() {
		events := make(chan window.Event, 256)
		w.Notify(events, window.AllEvents)

		for event := range events {
			e <- event
		}
	}()

	// Loop
	for {
		select {
		case event := <-e:
			if _, ok := event.(keyboard.StateEvent); ok {
				//log.Println(ev)
			} else if ev, ok := event.(keyboard.TypedEvent); ok {
				log.Println(ev)
				text.Runes = append(text.Runes, ev.Rune)
				text.Update()
				b := r.Bounds()
				text.Object.SetPos(lmath.Vec3{0, 0, float64(b.Dy()) / 2.0})
				r.Clear(image.Rect(0, 0, 0, 0), gfx.Color{0.2, 0.2, 0.2, 1})
				r.ClearDepth(image.Rect(0, 0, 0, 0), 1.0)
				r.Draw(image.Rect(0, 0, 0, 0), text.Object, camera)
			}
		default:
		}
		r.Render()
	}
}

func main() {
	window.Run(gfxLoop, nil)
}
