/*
 * File: zipwriter.go
 * Project: worker
 * File Created: Monday, 29th November 2021 9:45:00 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package archive

import (
	"archive/zip"
	"context"
	"io"
	"log"
	"path"

	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	errgroup "golang.org/x/sync/errgroup"
)

type FakeWriterAt struct {
	w io.Writer
}

func (fw FakeWriterAt) WriteAt(p []byte, offset int64) (n int, err error) {
	// ignore 'offset' because we forced sequential downloads
	return fw.w.Write(p)
}

type ObjectInput struct {
	Bucket *string
	Key    *string

	RawBytes map[string][]byte // filename->file bytes
}

type ObjectOutput struct {
	Bucket *string
	Key    *string
	//lint:ignore U1000 ignore, required for io.Pipe
	body io.Reader
}

type ZipWriter struct {
	downloader *manager.Downloader
	uploader   *manager.Uploader
}

func New(downloader *manager.Downloader, uploader *manager.Uploader) *ZipWriter {
	return &ZipWriter{
		downloader: downloader,
		uploader:   uploader,
	}
}

// zipS3Files is a function for streaming data from a bucket to a zip.Writer and then back to a bucket without utilizing disk.
func (z *ZipWriter) ZipS3Files(in <-chan *ObjectInput, out *ObjectOutput) error {
	var (
		totalBytes     int64
		uploadLocation string
	)

	// Create pipe
	pr, pw := io.Pipe()

	// Create zip.Write which will write to pipes
	zipWriter := zip.NewWriter(pw)

	// Create a cancellable context.
	// The downloader and uploader use the context to determine if the other has failed.
	// If either has failed, the context will be cancelled, signalling cleanup for the other.
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	// Run downloader
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		// We need to close our zip.Writer and also pipe writer - zip.Writer doesn't close underlying writer.
		// Closing the pipewriter lets the uploader know there are no more files to include in the archive,
		// allowing it to finish and return.
		defer func() {
			zipWriter.Close()
			pw.Close()
		}()

	loop:
		for {
			select {
			// Context cancelled
			case <-ctx.Done():
				log.Printf("exiting downloader")
				return nil
			case file, ok := <-in:
				// In channel closed i.e. no more data!
				if !ok {
					break loop
				}

				if file.RawBytes != nil { // file passed in directly
					// Sequentially download each file to writer from zip.Writer
					for filename, fileBytes := range file.RawBytes {
						log.Printf("reading file: %s", filename)

						w, err := zipWriter.Create(path.Base(filename))
						if err != nil {
							return err
						}

						n, err := w.Write(fileBytes)
						if err != nil {
							return err
						}
						log.Printf("read %d bytes", n)
						totalBytes += int64(n)
					}
				} else { // file needs downloaded
					log.Printf("downloading file: %s", *file.Key)

					// Sequentially download each file to writer from zip.Writer
					w, err := zipWriter.Create(path.Base(*file.Key))
					if err != nil {
						return err
					}

					s3File := s3.GetObjectInput{Bucket: file.Bucket, Key: file.Key}
					n, err := z.downloader.Download(ctx, FakeWriterAt{w}, &s3File)
					if err != nil {
						return err
					}
					log.Printf("downloaded %d bytes", n)
					totalBytes += n
				}
			}
		}
		return nil
	})

	g.Go(func() error {
		log.Printf("starting uploader")
		// Upload the file, body is `io.Reader` from pipe
		result := &s3.PutObjectInput{
			Bucket: out.Bucket,
			Key:    out.Key,
			Body:   pr,
		}

		upload, err := z.uploader.Upload(ctx, result)
		if err != nil {
			log.Printf("zipwriter upload error; err=%s", err.Error())
			if err == context.Canceled {
				return nil
			}
			return err
		}
		log.Printf("uploaded successfully; exiting uploader")
		uploadLocation = upload.Location
		return nil
	})

	// Wait for uploader and downloader to finish
	err := g.Wait()

	log.Printf("uploaded %d bytes to %s", totalBytes, uploadLocation)

	return err
}
