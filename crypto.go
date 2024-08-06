package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

func newEncryptionKey() []byte {
	keyBuff := make([]byte, 32)
	io.ReadFull(rand.Reader, keyBuff)
	return keyBuff
}

func copyDecrypt(key []byte, src io.Reader, dest io.Writer) (int, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}

	//read the iv from the reader
	iv := make([]byte, block.BlockSize())
	if _, err := src.Read(iv); err != nil {
		return 0, err
	}

	var (
		buff   = make([]byte, 32*1024)
		stream = cipher.NewCTR(block, iv)
		nw     = block.BlockSize()
	)

	for {
		n, err := src.Read(buff)
		if n > 0 {
			stream.XORKeyStream(buff, buff[:n])
			nn, err := dest.Write(buff[:n])
			if err != nil {
				return 0, err
			}
			nw += nn
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
	}

	return nw, nil
}

func copyEncrypt(key []byte, src io.Reader, dest io.Writer) (int, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}

	iv := make([]byte, block.BlockSize()) //16byte
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return 0, err
	}

	// prepend the iv to the file
	if _, err := dest.Write(iv); err != nil {
		return 0, err
	}

	var (
		buff   = make([]byte, 32*1024)
		stream = cipher.NewCTR(block, iv)
		nw     = block.BlockSize()
	)

	for {
		n, err := src.Read(buff)
		if n > 0 {
			stream.XORKeyStream(buff, buff[:n])
			nn, err := dest.Write(buff[:n])
			if err != nil {
				return 0, err
			}
			nw += nn
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
	}

	return nw, nil

}
