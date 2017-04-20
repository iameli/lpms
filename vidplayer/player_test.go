package vidplayer

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"

	"time"

	"net/url"

	"github.com/kz26/m3u8"
	"github.com/livepeer/lpms/stream"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/av/avutil"
	joy4rtmp "github.com/nareix/joy4/format/rtmp"
)

func TestRTMP(t *testing.T) {
	server := &joy4rtmp.Server{Addr: ":1936"}
	player := &VidPlayer{RtmpServer: server}
	var demuxer av.Demuxer
	gotUpvid := false
	gotPlayvid := false
	player.RtmpServer.HandlePublish = func(conn *joy4rtmp.Conn) {
		gotUpvid = true
		demuxer = conn
	}

	player.HandleRTMPPlay(func(ctx context.Context, reqPath string, dst av.MuxCloser) error {
		gotPlayvid = true
		fmt.Println(reqPath)
		avutil.CopyFile(dst, demuxer)
		return nil
	})

	// go server.ListenAndServe()

	// ffmpegCmd := "ffmpeg"
	// ffmpegArgs := []string{"-re", "-i", "../data/bunny2.mp4", "-c", "copy", "-f", "flv", "rtmp://localhost:1936/movie/stream"}
	// go exec.Command(ffmpegCmd, ffmpegArgs...).Run()

	// time.Sleep(time.Second * 1)

	// if gotUpvid == false {
	// 	t.Fatal("Didn't get the upstream video")
	// }

	// ffplayCmd := "ffplay"
	// ffplayArgs := []string{"rtmp://localhost:1936/movie/stream"}
	// go exec.Command(ffplayCmd, ffplayArgs...).Run()

	// time.Sleep(time.Second * 1)
	// if gotPlayvid == false {
	// 	t.Fatal("Didn't get the downstream video")
	// }
}

func TestHLS(t *testing.T) {
	player := &VidPlayer{}
	s := stream.NewVideoStream("test")
	s.HLSTimeout = time.Second * 5
	//Write some packets into the stream
	s.WriteHLSPlaylistToStream(m3u8.MediaPlaylist{})
	s.WriteHLSSegmentToStream(stream.HLSSegment{})
	var buffer *stream.HLSBuffer
	player.HandleHLSPlay(func(reqPath string) (*stream.HLSBuffer, error) {
		//if can't find local cache, start downloading, and store in cache.
		if buffer == nil {
			buffer := stream.NewHLSBuffer()
			ec := make(chan error, 1)
			go func() { ec <- s.ReadHLSFromStream(context.Background(), buffer) }()
			// select {
			// case err := <-ec:
			// 	return err
			// }
		}
		return buffer, nil

		// if strings.HasSuffix(reqPath, ".m3u8") {
		// 	pl, err := buffer.WaitAndPopPlaylist(ctx)
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	_, err = writer.Write(pl.Encode().Bytes())
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	return nil, nil
		// }

		// if strings.HasSuffix(reqPath, ".ts") {
		// 	pathArr := strings.Split(reqPath, "/")
		// 	segName := pathArr[len(pathArr)-1]
		// 	seg, err := buffer.WaitAndPopSegment(ctx, segName)
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	_, err = writer.Write(seg)
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// }

		// return nil, lpmsio.ErrNotFound
	})

	// go http.ListenAndServe(":8000", nil)

	//TODO: Add tests for checking if packets were written, etc.
}

type TestRWriter struct {
	bytes  []byte
	header map[string][]string
}

func (t *TestRWriter) Header() http.Header { return t.header }
func (t *TestRWriter) Write(b []byte) (int, error) {
	t.bytes = b
	return 0, nil
}
func (*TestRWriter) WriteHeader(int) {}

func TestHandleHLS(t *testing.T) {
	testBuf := stream.NewHLSBuffer()
	req := &http.Request{URL: &url.URL{Path: "test.m3u8"}}
	rw := &TestRWriter{header: make(map[string][]string)}

	pl, _ := m3u8.NewMediaPlaylist(10, 10)
	pl.Append("url1", 2, "url1")
	pl.Append("url2", 2, "url2")
	pl.Append("url3", 2, "url3")
	pl.Append("url4", 2, "url4")

	testBuf.WritePlaylist(*pl)

	handleHLS(rw, req, func(reqPath string) (*stream.HLSBuffer, error) {
		return testBuf, nil
	})

	p1, _ := m3u8.NewMediaPlaylist(10, 10)
	p1.DecodeFrom(bytes.NewReader(rw.bytes), true)
	segLen := 0
	for _, s := range p1.Segments {
		if s != nil {
			segLen = segLen + 1
		}
	}

	if segLen != 2 {
		t.Errorf("Expecting 2 segments, got %v", segLen)
	}
}