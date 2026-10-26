package app

/*
#cgo pkg-config: gtk4
#include <stdlib.h>
#include <gtk/gtk.h>

// gotk4 v0.3.1 cannot wrap GskRenderNode (not a GObject), so the
// snapshot -> node -> texture -> PNG chain lives in C.
static int wl_shot_png(uintptr_t wp, const char *path, int width, int height) {
	GtkWidget *w = (GtkWidget *)wp;
	GdkPaintable *p = gtk_widget_paintable_new(w);
	GtkSnapshot *s = gtk_snapshot_new();
	gdk_paintable_snapshot(p, s, width, height);
	GskRenderNode *node = gtk_snapshot_free_to_node(s);
	if (!node) { g_object_unref(p); return 0; }
	GskRenderer *r = gtk_native_get_renderer(gtk_widget_get_native(w));
	graphene_rect_t rect = GRAPHENE_RECT_INIT(0, 0, width, height);
	GdkTexture *tex = gsk_renderer_render_texture(r, node, &rect);
	int ok = tex ? gdk_texture_save_to_png(tex, path) : 0;
	if (tex) g_object_unref(tex);
	gsk_render_node_unref(node);
	g_object_unref(p);
	return ok;
}
*/
import "C"

import (
	"unsafe"

	coreglib "github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func savePNG(w gtk.Widgetter, path string, width, height int) bool {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	obj := coreglib.BaseObject(w)
	return C.wl_shot_png(C.uintptr_t(obj.Native()), cpath, C.int(width), C.int(height)) != 0
}
