/*
 * File: heatmap_test.go
 * Project: image
 * File Created: Sunday, 16th April 2023 6:11:26 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package image

import (
	"encoding/base64"
	"image"
	"image/jpeg"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"
)

const b64Image = "/9j/2wCEAAgGBgcGBQgHBwcJCQgKDBQNDAsLDBkSEw8UHRofHh0aHBwgJC4nICIsIxwcKDcpLDAxNDQ0Hyc5PTgyPC4zNDIBCQkJDAsMGA0NGDIhHCEyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMv/AABEIAHcAyAMBIgACEQEDEQH/xAGiAAABBQEBAQEBAQAAAAAAAAAAAQIDBAUGBwgJCgsQAAIBAwMCBAMFBQQEAAABfQECAwAEEQUSITFBBhNRYQcicRQygZGhCCNCscEVUtHwJDNicoIJChYXGBkaJSYnKCkqNDU2Nzg5OkNERUZHSElKU1RVVldYWVpjZGVmZ2hpanN0dXZ3eHl6g4SFhoeIiYqSk5SVlpeYmZqio6Slpqeoqaqys7S1tre4ubrCw8TFxsfIycrS09TV1tfY2drh4uPk5ebn6Onq8fLz9PX29/j5+gEAAwEBAQEBAQEBAQAAAAAAAAECAwQFBgcICQoLEQACAQIEBAMEBwUEBAABAncAAQIDEQQFITEGEkFRB2FxEyIygQgUQpGhscEJIzNS8BVictEKFiQ04SXxFxgZGiYnKCkqNTY3ODk6Q0RFRkdISUpTVFVWV1hZWmNkZWZnaGlqc3R1dnd4eXqCg4SFhoeIiYqSk5SVlpeYmZqio6Slpqeoqaqys7S1tre4ubrCw8TFxsfIycrS09TV1tfY2dri4+Tl5ufo6ery8/T19vf4+fr/2gAMAwEAAhEDEQA/APTQtUCu/UeP7w/z+laZwiFj0AzVO0QPc7+c8fyz/WuFI9VstFc3EY9OacRu1CJf7qFv6U9VzefRaWFd2oXDf3UVR+pppCbHTDJQe+aeozet/sp/WiT/AFyD2/rToPmuZz6EL+lUQ2LGN1y7egxSWwwZpe7HNOg/1Mz/AO036U+FcQt9QP5U0Jsfbr5cTr6E/wCNN2fNYqegJY/gP/r1NjCTn0z/ACrkfGnjKw8LW9s0p8y7kgJihXrk4wT6DrzVENkureL9I8ODVI7+6VZGnzHFGNztlRk4H865qX4xQfb1e20uWa2xtEjsFb8hn9a8cmnvNd1We6d2Z5WLOc9PwHSuk0rQJi0bvjaWwGH8XHf/ADitI07mbmelL8TY3vUmaxkwEYKCR1OO4znoe1RT/FAQyR3M1t+6X5W2clcn16Ht+GawbbQwsZk+bAUYOMc4+8P0qO/0CKaGXKkJIgDDHTj+ec1r9XM/bo7K6+KWiPqNrbxea+JFzLsIBJ7AfjXcPMGWC/COqKSsgYYIUnGT9CAa+Z9Mgk0bXba4ZVmYB1wRkKcEBh7/ANa9TX4t2lno8cD232m8CbXBwif1yKxnFw1ZcXzWSPULyJJrGeN2AR42UkngAjrXzh4y8QzeIr63jYDy7GIQ7icgsOCR65xVrVfG2qavEYJrp4rYjCwo2FAz0Pc/jXE3VwPOdSwIzhaxc3N7WNlDkWrIrmY7SEIJHHFSWkMksiGTckSkbnVc7c9KiWzmmlxEpfIzxXdaVLZW/h+CWG3tpJ/KYyAxbgXBGN2fSiTSQ4xbd2YL2wtraZJ1LOejnggVQs7VYh5g4MgOwDsPX8atXly1/clFwA2c47etOfKR7QoU44x6VzXaJk1J37EYZlUKuCOh9aqQbsyzseWOc+w4H+P40+aRXCjA3njjr71A8xA2JkDp9BVJEX1Oh02QzZK4DMpxuGeau+Rc+sH/AHyayNLmWMoGPB6+1bPnw/8APUVOi0M6ibd0ey3AEzLbZ+/y2P7op0OBqMqjgKi4H1//AFVxJu7oS+YYp2x6HH9atw312kjSq8wYjBA9q6uVnVznZp/x/N/uD+dFtxfXf/Af5VzZ1e5UrIk4D7cNuHNOh8RyRTvI6xnzMA9cjHeizDnTOmPN4B6Lmn2fKyN6yNWDH4ij+1tIEDbgBnOMVcstahQMkkbDLsQRgjBNILmtbJ/ojDuS38zT7cbrUn3JrPsdYso7ZUlm2sM5JHHX1rn7nxuttCYbGMSBWbfK3Tr/AA461SJZ1eoX8GmaLeahOQIoo3kPvxwK+bTFfeJdZluLnzZp5yGZs9B2HPQAdq6jxLruoalHb6bNOSJn3GNR0Uc8/iasWF/pmj2gQskcY5eQ8lj/AFq4u2rIcbk+keBrWLy5JNxcH+ByNvt9K7Gz0m2gUIsSArjBx39eK5iLxxokSGQX0ewduh/KtSLxLaTwG5inXydu4sTXTBWV2YTd3ZHRSQW6wsSiKoO7I9RUMljBcRPxmPnADcGuF1PxtFcyrEnzRAYJP8VdT4V1u3vrNkjlJaM9C3IH0raMk3YwlBpXOd8ReHRFbySRwhVyWLN3Ppz2rzXWS9vchgAQp2sE5xx3FfRsttHfRlGBZT9B9a8m8c+DjbXDT2yjyz0PXb7VVSKcSacmmefPKJlznCmo1iZ5tsQ3k80QwO0ptwjF0O3GOldl4c0aW2u4ibd2k3MknGccfjjr+NeVUfI7HqU1zrmZoaXoa6bHJbhVuJ5oFmyOQgHOOPWs+8i+x2jWCw+VcNIzHgD92cYrdSbULa1054TJCXV4JW6g7cjo3fiuQ1Ys2rSfvHJiC7mY8n5R6Vzy1NJu0Sq5W0jZQMux+Y+1QLMJBsJAIOFJNVp5y7bWTYOuQ2Q340sUZlj2gjGOvp601Gyuzkvd2Q4/vDuwfTco6ioZ2RV2KDux3NLNdiIBF5UccVUMnmOPmBBPOOpq4pktluC4ZX46jpzxVz7bJ6J/31VABCAyDijA/uf+P0nFMq57wZPLKkRBlboTSwzM7kLEgB/vcU4rGvDPgqxAGeaeTESQwx/wKtlJ9TVpEcjbZiFjiJ+uKQJKZCDbqcjqGGKm2Bw2zoQCaemwHaWIYcAbqLhYr/ZYopQPI355wMVFdta2sZefzIUzgEHoas6lNa2EDXMr42jhO7H0FcVdXcupTGaY4X+H0UHsPei7CwXt+95IVhdvKzhfU/hVO+u1sIFZ5Cz9FTrz/jTrmdLeAJGBGMfNjqR7muVubpnklkZiSnyp6An0/DiplK2iKjG5n6lrc8U0zxttmk+Tf3A74rGN9c3Z2SzSSFu2c1ftrNr28ZyFYejHArpI47S0t/3NrEkg6uF5rRSUUZSi5SMOx8PO6C4vAYoBzycFvpVrVNXlEUVpHkQjkqveoLzVLmScRBGkHp1NZkk9wZy5Rwy/7PStU+Yya5di5ZanHJMYnSQbG+Yd8Vt6P4ih0zU47uwuSwjcB0YY3LnkY+lcoryiR5TC4z95th59zVuBbRRF5cfltwGIOc+9bxdkYu7Z6frPxKu9F8QyQEKbJ8GN0HzAVv2nijTvEumkK6Oejq/WuBvtOh8R2PmW0ireQr0bkSgDn8a5jRdZGg3UhZd4Y/MuMEH2qnNp3HGMbao65LUQeItVtkjQp9n3iRl3YGR8w98Vp3KpFKslvdShUmicMvyrtZcEnnPaoPD91Z6xe3l15ZeGW38sb+Cp64NWpfPksNyxwRqbZWJ2gZCP1HvzXmYnWo7HpUP4aJbSNzrcbJE0i/aJkV9uRlgD3rkdXm339xJnLO/OB6DGP0rpdanltorm4WTkSxyp8uM70xxj6GuFlnBEnmtsUHOc4BrFK5NeVlYbKilv6VSW9UmaOMjA+XPYmqt7qhZTHAODxvPX8KqWdwkX7uUHyyc5Fbxpu12cbl2Lyp8xZiCRzgGnKvl8nCluMY5NP3KYf3BV89NlQrG5ffJk46ZOae4i/AoEfHJ+vFO5/wBmoFcqhbIwuOpxxSfb4/70X/fVZ2ZVz394lk6oAZF+Xb2I96dDauyAiX5iOSwPUU6RGEkhLsdp3Lt/WlSeRMoSxyNykYxWljqFZsKikDqV4XrSnbFEXeNB8pyTnjFJv3iQ7nLD5goHA/GsPxBrTLGbCJy3mAM3sD2z70tw2MnWNQk1O8UrgQp8sY7H1OKz57i3iQb1Msg5+98oPue1O8qW7lMUAeQYyxUcCk1DRZYbMs7zcDO3AH8+K05WyboydT1G3Kr5YaSRh1PCr+H+Nc1czhYOPvFdx5yTn1qLUJZYGlRgwLDoxyT9arl98QB+9swfrgVHJrcrmtoT6ZfC3kAYAjPGR0ree6PlAttwew7Vw5lIlJOcg84NbVtdl0ypzkc8Zq5QMVM1zaLKwlhA3Y5x1rOurdw7EZGDzmrkN/5OQMkgck1FchLn5oiOTk5Ocmrptx0FNKWpVS3uoVVonYnHIGakW184gyQDr94cE1LAk6HDMQPXNdTor2qo4njSVw2cn2rpg1LQ55Ra1MzSrG80668zypMOflDHORUniexsZYZLtLby5OC5BxzXRXMrSgOBkD06H/CuY8RakssBt02MXbHmA8j6+tDg4XY1Lmsbvw6hWz0qS5MqbXuF+Ukk4Cn0B9aszKoRrpYimTN5YfLY4BBAP9aPCLTWGjIgtA8nmJJl2KqAcj19c04Xvl2jyqbO1dZpiWkk3ZOw9R+gryKkryPVpxtFHO+JNZN5cxfaZPmht0V2zkM2D+vNcVfXX2h/lUrGv3QaLy+lvLgySEkkkhfTNVyDXRCHLqzzqlTmZXYdzTCB071K/UAVGwwTWyMiLLIxKsQfUGpUv7pePM3D0YA1HIOc+1JGhZqdk1qBca7eaAoUVQTliO9RfJ/sUSY27R09aj2+9SkgPpqC93WynfLI6n5nGSCPfNOaZkERCu+zoQe1ZT6hp+lu/nTqiyjKIJOWPv6965PWvFt3cYWycRQdAxf5iPoOlYpX2PQOz1PU100K8mYivZpOoPTiuLXU4BM0zQyTMeTngH8z2rnZLyWZ/wB5Jk4zkj/GmS3jWw3eac46GrhTa1FKaOzj1l3t2WJ57SIHOE2qPxPOaxdS12BHfZId3Z3fcc+2f6Vx1zrV3ONiSSMOxGD/AEptqZpC7zh+Dj5uCc1s3oYxabEu3aWcOQSXGTnrycVXVytwynoRW0bdZhE7LgthRjsOlZd5CTaW90o++Mn2PJI/MGosXLR3My5UiQkfdJyKI5zGQVOBnoe9aKQi7gJUgMPXoaz2tpUkK4we9UmYyjbU0Yb1SoVwQx6mrS3CkZLDA/CsI7kIzkjrQHfAw3X0p2Fc3lumkYHcikepq1BqK28+fOXGcHB5rmYw8xOWJwQOferAtXUMQOQcdapaaibudHq3iXzLL7NaCRX6s/A49BWfpEFxqVzEqhnJbovWqMVnLNMMg/M2Poe4ru9I0dIltyZrXZGHaVTNGMdfX1/Gs69Z2v1NKFK78jc021xmJoWm8sRKBKjsfX+E46eprGuPIvUnjMcXzB3MUayfKc4AwOnT1xVmPWNNiklaWVGcghY1gQr0wM8gevOM1j3usQWNrNHBkvIqqDFO+B3Pt36V5yi29j0G0lqYut6bZ2LxG1O0soLKZhIc4znjp9KxXOBxU91ezXcwaQ/KOAPQVWfqBXZFNLU8yslzXitCIg8tTX+7mpsVE4yrj0qzIhcZAqRRg7B+Jq7aWCPBLcXEiIqDaiF9rMxBIIGOQO/1quibF3U7jaaI3Q/hTMCpmfcBTcf7I/OgR2tx+/fzZXw+c/KM5/M01bOJsthz3GOSfWobu+ntwqiNSTkHbNjP44qEanJHbK8MI3MSApk3Z6H7xAz17VcUkdMpF2SNzlIEKewySBVJ7IYLSyZKAk/MDUENxd3rSRl3gkjO8oCcEd+PX3+oqzchYopIVGwsoVfTvx+Qq2rkqRQt7RwN6sR84OR/npVyCHyw0hAwyh2XuCOf1pWzBA687gxAXPbj/CqouS7oP4cE49c1NkirmgZMrCyhSQWGB0OMH+n61nTHfaeSx5iY89vX/GoY7oxbVOSFBH41XMxYkbsg9D9KlhdFeOY2swY5GDggdxnp+VbYW3vIVdNhJHHFUINPN2Nx4GcDjk1es9NYArGxwcsPTA601G7Fey8ihLpsocxhc/T+lVp7Vox8qYHauiSHap+fcCAVPrSy2vmhlIBPHGO3Y1qoaXMnvoYen23msQAS4ORjvVowS+YAQox3PStbT7JICcknI+bHap7u2QDapAJHBT+dP2TcbiU0nZmYjSQhXhb5VOceh6GqhmlPKDBz1NPljmiP3sc9R0NNEqsRwA/txmuWcTshK60FYyM25nznqKhdgW24qaV9gDEVWRTLJkYAH6VCLl2IvLLsflzjtV1NN82AHd85PAA4H/1/arGnWzXWoxW0dtLcbidyRY3EAZP6Cr81xbNPJ9k8jY7+XDDJKRJGoxkYXjLcjrnmiT6GbimW9I8NaRDp91qWsObpIIQ32a3uliPmE4CknknHOFzV7TtN0XTfD0Vz5OmXmqyuGRGu9zqvUnyyMBhg4yfwq5fpqEljY+G4tP1sMGF5d2zwR52g/KYz98gc43HsKwte1I+I9YU3N4tuIyLYRXsPMAHJbEaAdQB3NZ3b3YKEU9EW9a1++udBljmmvpJL4FYI5xE6+WSNxyoyG+UenSuVTS5kVpi9u8cahnVZQWx3G04JI7gVfitGsL43HlzpvJNrd2iMkZI6kbl5A71c+2CWLEria3Qh0MkaBt/fkdef/wBVJy5VoN0lN6jdJ0TSbmz82fT5vMZSAZZvkbJ+8FGGX8yKs/8ACM6R/wA+sf8A39f/AOKrHutannlKQhiucZHAFQ/arz1/U/4VP7163sWo0lolcdbW6qIxdSl3VWY71O7nI/l71pW1sksYSJ0Yxg5jK5B56/XA/Snwxpd2cJYgzFCqynj8G/Olhm+y3UXBG7CuMcjnGa9ONrXONp3sVApt5D5RJ/iAZegJ5H0pl2G88gAlVCuuR2I4rTKRxzbkJXytybRwDhQP15qrcSh1VOdxxx6DFQ2aRhZalW92iJZR1Ay2OO5rPCs8uEwR2J9K1mt/McqUUKO1LHbR+aIx8qnk46kU1FsUmY0tvtDhckZwDjqauWeljfGZOSTkj+6Byf6fnWlJGjMzqi4QfID6+v5/yp8QzsDZySWb37f402khRXMSJAqwyFQFCjA/Hn/CpoYVQT/IQVBBx7gDFP8AllDNtZkBBC9M4PH+fenHzJDKUHLMEBx0Pc/rQtzV7GckIjtkRgTgAZB/D+lPgfzMHjeARWpZWCtIqTgqueB6cH/Gq9wIUuZViHGcoParvomZcvvNEKId8jcKeuM9f8iq87kRqS29d3ynutXLiaPyUKgnb94Z7H/JrNaVFmKE8Hj6e9OTsiFG71Kc7ozHIAPfjGT71nTIVPBLD6cir87IwPGHU4IXoazp2wu4c4rlk9TpSSQsU4P7tsN/sn+lSrE2792Dz271S83gHqx9eoFWI3kZOVYkdDWbVhpp6HQrpogti8bSbJk3R/a4Vj3R4yzK+cZ3AqMc1qafpsjwme5nsrKNFDiLUWLrMgG5EXaucge4JzXPWF5cpG0OA6EjcrqGAA5HXoPpWrfaqX0OHSbWS7S2JElwkpjZXfP8GF3AfUmsXvqbKN1oRjxII3uZfs8YubhtylZpP3K8jYuTyvPcnpWa13PPiRnLAfLkgZAznnv1NMmijRBsXAHJPfHehGXyjkheCSw6DHT9Kem6LUbOzNGDVNRt7gSw3txBIoIVo5D8gPoeuDUGVKsG5LNliTknPf8AE1V85PKDKOMcnPvUJuvlAUjrnjvS5WyuaK1LR2IAoXAA/wA/zpm9fU/981A83yEjnkgfWofNl/2apRJdRLY6CwfdhhgRbmBGPoM/zpJWxIzMMuhA6+9M0z/j0P8Avt/MUtx/rJv+ug/nXWzkitEMubtUlkAJyGPPv1qqs5OHYHc3HH+f85pl5/rpv99v5U0fdj+p/pTREm72NFFIiDuPlz69farEy/ZW3MoMjjI/xP8AhUTf8eI+rf0qbVP9bD/1z/xrV6IhasqhwIwrZz/rGOPToKnhw+McKoGfU/5/rVR+jf8AXP8ArVm1+4/+7/hWbNYlp5fKhHUvIc9ewrQtrpEB3LyFyfr61l3P3Yf9w/zqynWT/cNOL3KktUiS7ncu+GwEj4I9T/8AqrIN0ck/xBDye/IyP1rRuesv/XNf51it95/o38xU3HNWsSSXbAlSPlPDD1wOf51TupCrKQeo549KfP8A6z/gTfyqK86D6GiT0MVuVwGk3PnblceuKma2UkhiMZ4wOcUyL/Un6f41afr/AJ9a5ptnXSimtSNLEsCUVVXOSW7CpFtraIBnkdzweOBVuD/j3b6VUl/1Q+i1mm2auKSvYe13CsW+LAB45B61V+1rI42sRt4zjrUI/wCPIf71QQdT/vGq5ET7R3RPc3PzmBAflxuPqe34U4QsEVWc46nn7xP4VVl/4/ZfwrQk6p9P6Gm9LIhPmbbKD3duymGIDaBnLbqcsW6FARuOTjDYrLj/ANY3+6f51swfdT8f5Vcly7GVOXO9Ru35DhCVXrsbBH51Hvj/ALk/5rU8f+qnqvSQ5O1j/9k="

// TestHeatmap simply tests that we can generate a heatmap. It does not validate the accuracy of it.

func TestHeatmap(t *testing.T) {
	reader := base64.NewDecoder(base64.StdEncoding, strings.NewReader(b64Image))
	img, _, err := image.Decode(reader)
	if err != nil {
		panic(err)
	}

	boundingBoxes := []BoundingBox{}
	rand.New(rand.NewSource(time.Now().UnixNano()))

	b := img.Bounds()
	for i := 0; i < 10; i++ {
		xmin := rand.Intn(b.Dx()-0+1) + 0
		xmax := rand.Intn(b.Dx()-xmin+1) + xmin
		ymin := rand.Intn(b.Dy()-0+1) + 0
		ymax := rand.Intn(b.Dy()-ymin+1) + ymin
		boundingBoxes = append(boundingBoxes, BoundingBox{
			Xmin: xmin,
			Xmax: xmax,
			Ymin: ymin,
			Ymax: ymax,
		})
	}

	points := []DataPoint{}

	for _, box := range boundingBoxes {
		points = append(
			points, P(
				float64(box.Xmin),
				float64(box.Ymin),
				box.Xmax-box.Xmin,
				box.Ymax-box.Ymin),
		)
	}

	heatmap := Heatmap(img.Bounds(),
		points, nil, 100, Classic, OverlayShapeDot)

	overlay := AddOverlay(heatmap, img, 0.5)

	file, err := os.CreateTemp(os.TempDir(), "emld_heatmap.*.jpeg")
	if err != nil {
		panic(err)
	}
	jpeg.Encode(file, overlay, &jpeg.Options{Quality: 100})

	t.Logf("Created heat map at: %s", file.Name())
}
