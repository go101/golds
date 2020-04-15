package m

type ByteScanner interface {
	ByteReader
	UnreadByte() error
}

type ByteReader interface {
	ReadByte() (byte, error)
}
