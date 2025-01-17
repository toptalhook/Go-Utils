package crypto

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
	"math/big"
	"testing"

	"github.com/Laisky/zap"
	"github.com/stretchr/testify/require"
	"go.dedis.ch/kyber/v3/group/edwards25519"
	dediskey "go.dedis.ch/kyber/v3/util/key"

	"github.com/Laisky/go-utils/v4/log"
)

func TestPassword(t *testing.T) {
	t.Parallel()

	password := []byte("1234567890")
	hp, err := GeneratePasswordHash(password)
	require.NoError(t, err)

	t.Logf("got hashed password: %v", string(hp))

	require.True(t, ValidatePasswordHash(hp, password))
	require.False(t, ValidatePasswordHash(hp, []byte("dj23fij2f32")))
}

func ExampleGeneratePasswordHash() {
	// generate hashed password
	rawPassword := []byte("1234567890")
	hashedPassword, err := GeneratePasswordHash(rawPassword)
	if err != nil {
		log.Shared.Error("try to generate password got error", zap.Error(err))
		return
	}
	fmt.Printf("got new hashed pasword: %v\n", string(hashedPassword))

	// validate passowrd
	if !ValidatePasswordHash(hashedPassword, rawPassword) {
		log.Shared.Error("password invalidate", zap.Error(err))
		return
	}
}

// goos: linux
// goarch: amd64
// pkg: github.com/Laisky/go-utils/v4/crypto
// cpu: Intel(R) Xeon(R) Gold 5320 CPU @ 2.20GHz
// BenchmarkGeneratePasswordHash
// BenchmarkGeneratePasswordHash/generate
// BenchmarkGeneratePasswordHash/generate-104         	       1	1256584728 ns/op	   19120 B/op	      16 allocs/op
// BenchmarkGeneratePasswordHash/validate
// BenchmarkGeneratePasswordHash/validate-104         	       1	1255534569 ns/op	   19216 B/op	      18 allocs/op
// BenchmarkGeneratePasswordHash/invalidate
// BenchmarkGeneratePasswordHash/invalidate-104       	       1	1253798232 ns/op	   19216 B/op	      18 allocs/op
func BenchmarkGeneratePasswordHash(b *testing.B) {
	pw := []byte("28jijf23f92of92o3jf23fjo2")
	ph, err := GeneratePasswordHash(pw)
	require.NoError(b, err)

	phw, err := GeneratePasswordHash([]byte("j23foj9foj29fj23fj"))
	require.NoError(b, err)

	b.Run("generate", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err = GeneratePasswordHash(pw)
			require.NoError(b, err)
		}
	})
	b.Run("validate", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			require.True(b, ValidatePasswordHash(ph, pw))
		}
	})
	b.Run("invalidate", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			require.False(b, ValidatePasswordHash(phw, pw))
		}
	})
}

func TestECDSAKeySerializer(t *testing.T) {
	t.Parallel()

	priKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	// var (
	// 	priByte, pubByte []byte
	// )
	// if pubByte, err = EncodeECDSAPublicKey(&priKey.PublicKey); err != nil {
	// 	t.Fatalf("%+v", err)
	// }
	// t.Logf("pub: %v", string(pubByte))
	// if priByte, err = EncodeECDSAPrivateKey(priKey); err != nil {
	// 	t.Fatalf("%+v", err)
	// }
	// t.Logf("pri: %v", string(priByte))

	// var (
	// 	priKey2 *ecdsa.PrivateKey
	// 	pubKey2 *ecdsa.PublicKey
	// )
	// if _, err = DecodeECDSAPublicKey(pubByte); err != nil {
	// 	t.Fatalf("%+v", err)
	// }
	// if priKey2, err = DecodeECDSAPrivateKey(priByte); err != nil {
	// 	t.Fatalf("%+v", err)
	// }

	hash := sha256.Sum256([]byte("hello, world"))
	r, s, err := ecdsa.Sign(rand.Reader, priKey, hash[:])
	require.NoError(t, err)

	t.Logf("generate hash: %x %x", r, s)
	require.True(t, ecdsa.Verify(&priKey.PublicKey, hash[:], r, s))

	// t.Error()
}

func TestECDSAVerify(t *testing.T) {
	t.Parallel()

	priKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	priKey2, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	for _, plainLen := range []int{
		1, 1024, 10240,
	} {
		plain, err := Salt(plainLen)
		require.NoError(t, err)
		t.Run(fmt.Sprintf("plainLen=%d", plainLen), func(t *testing.T) {
			t.Parallel()

			t.Run("correct key", func(t *testing.T) {
				r, s, err := SignByECDSAWithSHA256(priKey, plain)
				require.NoError(t, err)
				require.True(t, VerifyByECDSAWithSHA256(&priKey.PublicKey, plain, r, s))
			})

			t.Run("incorrect plain", func(t *testing.T) {
				r, s, err := SignByECDSAWithSHA256(priKey, plain)
				require.NoError(t, err)
				require.False(t, VerifyByECDSAWithSHA256(&priKey.PublicKey, append(plain, '2'), r, s))
			})

			t.Run("incorrect key", func(t *testing.T) {
				r, s, err := SignByECDSAWithSHA256(priKey2, plain)
				require.NoError(t, err)
				require.False(t, VerifyByECDSAWithSHA256(&priKey.PublicKey, plain, r, s))
			})
		})
	}
}

func TestRSAVerify(t *testing.T) {
	t.Parallel()

	priKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	priKey2, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	for _, plainLen := range []int{
		1, 1024, 10240,
	} {
		plain, err := Salt(plainLen)
		require.NoError(t, err)
		t.Run(fmt.Sprintf("plainLen=%d", plainLen), func(t *testing.T) {
			t.Parallel()

			t.Run("correct key", func(t *testing.T) {
				sig, err := SignByRSAPKCS1v15WithSHA256(priKey, plain)
				require.NoError(t, err)

				err = VerifyByRSAPKCS1v15WithSHA256(&priKey.PublicKey, plain, sig)
				require.NoError(t, err)
			})

			t.Run("incorrect plain", func(t *testing.T) {
				sig, err := SignByRSAPKCS1v15WithSHA256(priKey, plain)
				require.NoError(t, err)

				err = VerifyByRSAPKCS1v15WithSHA256(&priKey.PublicKey, append(plain, '2'), sig)
				require.ErrorContains(t, err, "verification error")
			})

			t.Run("incorrect key", func(t *testing.T) {
				sig, err := SignByRSAPKCS1v15WithSHA256(priKey2, plain)
				require.NoError(t, err)

				err = VerifyByRSAPKCS1v15WithSHA256(&priKey.PublicKey, plain, sig)
				require.ErrorContains(t, err, "verification error")
			})
		})
	}
}

func TestRSAPSSVerify(t *testing.T) {
	t.Parallel()

	priKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	priKey2, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	for _, plainLen := range []int{
		1, 1024, 10240,
	} {
		plain, err := Salt(plainLen)
		require.NoError(t, err)
		t.Run(fmt.Sprintf("plainLen=%d", plainLen), func(t *testing.T) {
			t.Parallel()

			t.Run("correct key", func(t *testing.T) {
				sig, err := SignByRSAPSSWithSHA256(priKey, plain)
				require.NoError(t, err)

				err = VerifyByRSAPSSWithSHA256(&priKey.PublicKey, plain, sig)
				require.NoError(t, err)
			})

			t.Run("incorrect plain", func(t *testing.T) {
				sig, err := SignByRSAPSSWithSHA256(priKey, plain)
				require.NoError(t, err)

				err = VerifyByRSAPSSWithSHA256(&priKey.PublicKey, append(plain, '2'), sig)
				require.ErrorContains(t, err, "verification error")
			})

			t.Run("incorrect key", func(t *testing.T) {
				sig, err := SignByRSAPSSWithSHA256(priKey2, plain)
				require.NoError(t, err)
				err = VerifyByRSAPSSWithSHA256(&priKey.PublicKey, plain, sig)
				require.ErrorContains(t, err, "verification error")
			})

			t.Run("indetermistic sig", func(t *testing.T) {
				sig1, err := SignByRSAPSSWithSHA256(priKey, plain)
				require.NoError(t, err)
				sig2, err := SignByRSAPSSWithSHA256(priKey, plain)
				require.NoError(t, err)

				require.NotEqual(t, sig1, sig2)
			})
		})
	}
}

func ExampleSignByECDSAWithSHA256() {
	priKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Shared.Panic("generate key", zap.Error(err))
	}
	priKey2, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Shared.Panic("generate key", zap.Error(err))
	}

	// case: correct key
	cnt := []byte("fjijf23lijfl23ijrl32jra9pfie9wpfi")
	r, s, err := SignByECDSAWithSHA256(priKey, cnt)
	if err != nil {
		log.Shared.Panic("sign", zap.Error(err))
	}
	if !VerifyByECDSAWithSHA256(&priKey.PublicKey, cnt, r, s) {
		log.Shared.Panic("verify failed")
	}

	// generate string
	encoded := EncodeES256SignByBase64(r, s)
	if _, _, err = DecodeES256SignByBase64(encoded); err != nil {
		log.Shared.Panic("encode and decode", zap.Error(err))
	}

	// case: incorrect cnt
	cnt = []byte("fjijf23lijfl23ijrl32jra9pfie9wpfi")
	r, s, err = SignByECDSAWithSHA256(priKey, cnt)
	if err != nil {
		log.Shared.Panic("sign", zap.Error(err))
	}
	if VerifyByECDSAWithSHA256(&priKey.PublicKey, append(cnt, '2'), r, s) {
		log.Shared.Panic("should not verify")
	}

	// case: incorrect key
	r, s, err = SignByECDSAWithSHA256(priKey2, cnt)
	if err != nil {
		log.Shared.Panic("sign", zap.Error(err))
	}
	if VerifyByECDSAWithSHA256(&priKey.PublicKey, cnt, r, s) {
		log.Shared.Panic("should not verify")
	}
}

func TestFormatBig2Hex(t *testing.T) {
	t.Parallel()

	b := new(big.Int)
	b = b.SetInt64(490348974827092350)
	hex := FormatBig2Hex(b)

	t.Logf("%x, %v", b, hex)
	require.Equal(t, hex, fmt.Sprintf("%x", b))
}

func TestFormatBig2Base64(t *testing.T) {
	t.Parallel()

	b := new(big.Int)
	b = b.SetInt64(490348974827092350)
	r := FormatBig2Base64(b)
	require.Equal(t, r, "Bs4Ry2yLuX4=")
}

func TestParseHex2Big(t *testing.T) {
	t.Parallel()

	hex := "6ce11cb6c8bb97e"
	b, ok := ParseHex2Big(hex)
	require.True(t, ok)

	t.Logf("%x, %v", b, hex)
	require.Equal(t, hex, fmt.Sprintf("%x", b))
}

func TestParseBase642Big(t *testing.T) {
	t.Parallel()

	raw := "Bs4Ry2yLuX4="
	b, err := ParseBase642Big(raw)
	require.NoError(t, err)

	t.Log(b.String())
	require.Equal(t, "490348974827092350", b.String())
}

func TestECDSASignFormatAndParseByHex(t *testing.T) {
	t.Parallel()

	a := new(big.Int)
	a = a.SetInt64(490348974827092350)
	b := new(big.Int)
	b = b.SetInt64(9482039480932482)

	encoded := EncodeES256SignByHex(a, b)
	t.Logf("encoded: %v", encoded)

	a2, b2, err := DecodeES256SignByHex(encoded)
	require.NoError(t, err)

	require.Equal(t, 0, a2.Cmp(a))
	require.Equal(t, 0, b2.Cmp(b))
}

func TestECDSASignFormatAndParseByBase64(t *testing.T) {
	t.Parallel()

	a := new(big.Int)
	a = a.SetInt64(490348974827092350)
	b := new(big.Int)
	b = b.SetInt64(9482039480932482)

	encoded := EncodeES256SignByBase64(a, b)
	t.Logf("encoded: %v", encoded)

	a2, b2, err := DecodeES256SignByBase64(encoded)
	require.NoError(t, err)

	require.Equal(t, 0, a2.Cmp(a))
	require.Equal(t, 0, b2.Cmp(b))
}

// func Test_expandAesSecret(t *testing.T) {
// 	type args struct {
// 		secret []byte
// 	}
// 	tests := []struct {
// 		name string
// 		args args
// 		want int
// 	}{
// 		{"0", args{[]byte("")}, 16},
// 		{"1", args{[]byte("1")}, 16},
// 		{"2", args{[]byte("12")}, 16},
// 		{"3", args{[]byte("14124")}, 16},
// 		{"4", args{[]byte("1535435535")}, 16},
// 		{"5", args{[]byte("   43242341")}, 16},
// 		{"6", args{[]byte("1111111111111111")}, 16},
// 		{"7", args{[]byte("11111111111111111")}, 24},
// 		{"8", args{[]byte("11111111111111111   ")}, 24},
// 		{"9", args{[]byte("11111111111111111   23423 4324   ")}, 32},
// 		{"10", args{[]byte("11111111111111111   23423 4324   111")}, 32},
// 		{"11", args{[]byte("11111111111111111   23423 4324   111414124")}, 32},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			if got := expandAesSecret(tt.args.secret); len(got) != tt.want {
// 				t.Errorf("expandAesSecret() = (%d)%v, want %v", len(got), got, tt.want)
// 			}
// 		})
// 	}

// 	// race
// 	var pool errgroup.Group
// 	secret := make([]byte, 5, 10)
// 	for i := 0; i < 17; i++ {
// 		pool.Go(func() error {
// 			expandAesSecret(secret)
// 			return nil
// 		})
// 	}

// 	if err := pool.Wait(); err != nil {
// 		t.Fatalf("%+v", err)
// 	}
// }

func TestSignReaderByEd25519WithSHA256(t *testing.T) {
	t.Parallel()

	raw, err := Salt(100 * 1024 * 1024)
	require.NoError(t, err)

	prikey, err := NewEd25519Prikey()
	require.NoError(t, err)

	sig, err := SignReaderByEd25519WithSHA256(prikey, bytes.NewReader(raw))
	require.NoError(t, err)

	pubkey := Prikey2Pubkey(prikey).(ed25519.PublicKey)
	err = VerifyReaderByEd25519WithSHA256(pubkey, bytes.NewReader(raw), sig)
	require.NoError(t, err)

	t.Run("false pubkey", func(t *testing.T) {
		prikey, err := NewEd25519Prikey()
		require.NoError(t, err)
		pubkey := Prikey2Pubkey(prikey).(ed25519.PublicKey)

		err = VerifyReaderByEd25519WithSHA256(pubkey, bytes.NewReader(raw), sig)
		require.ErrorContains(t, err, "invalid signature")
	})

	t.Run("false sig", func(t *testing.T) {
		falseSig := []byte("2l3fj238f83fu")
		err := VerifyReaderByEd25519WithSHA256(pubkey, bytes.NewReader(raw), falseSig)
		require.ErrorContains(t, err, "invalid signature")
	})
}

func TestVerifyBySchnorrSha256(t *testing.T) {
	t.Parallel()

	suite := edwards25519.NewBlakeSHA256Ed25519()
	keyPair := dediskey.NewKeyPair(suite)

	content := []byte("hello, world")

	t.Run("pubkey marshal & unmarshal", func(t *testing.T) {
		pub, err := keyPair.Public.MarshalBinary()
		require.NoError(t, err)

		pub2 := suite.Point()
		err = pub2.UnmarshalBinary(pub)
		require.NoError(t, err)

		require.True(t, keyPair.Public.Equal(pub2))
	})

	t.Run("prikey marshal & unmarshal", func(t *testing.T) {
		priBytes, err := keyPair.Private.MarshalBinary()
		require.NoError(t, err)

		pri2 := suite.Scalar()
		pri2.UnmarshalBinary(priBytes)
		require.NoError(t, err)

		t.Run("sign & verify", func(t *testing.T) {
			sig, err := SignBySchnorrSha256(suite, pri2, bytes.NewReader(content))
			require.NoError(t, err)

			err = VerifyBySchnorrSha256(suite, keyPair.Public, bytes.NewReader(content), sig)
			require.NoError(t, err)
		})

		t.Run("sign & invalid verify", func(t *testing.T) {
			keyPair2 := dediskey.NewKeyPair(suite)

			sig, err := SignBySchnorrSha256(suite, keyPair.Private, bytes.NewReader(content))
			require.NoError(t, err)

			err = VerifyBySchnorrSha256(suite, keyPair2.Public, bytes.NewReader(content), sig)
			require.ErrorContains(t, err, "invalid signature")
		})

		t.Run("sign & invalid verify", func(t *testing.T) {
			keyPair2 := dediskey.NewKeyPair(suite)

			sig, err := SignBySchnorrSha256(suite, keyPair2.Private, bytes.NewReader(content))
			require.NoError(t, err)

			err = VerifyBySchnorrSha256(suite, keyPair.Public, bytes.NewReader(content), sig)
			require.ErrorContains(t, err, "invalid signature")
		})
	})
}

// goos: linux
// goarch: amd64
// pkg: github.com/Laisky/go-utils/v4/crypto
// cpu: Intel(R) Xeon(R) Gold 5320 CPU @ 2.20GHz
// Benchmark_Sign
// Benchmark_Sign/sign_rsa-2048_4k
// Benchmark_Sign/sign_rsa-2048_4k-104         	      98	  12063393 ns/op	     896 B/op	       5 allocs/op
// Benchmark_Sign/sign_rsa-4096_4k
// Benchmark_Sign/sign_rsa-4096_4k-104         	      22	  53844454 ns/op	   38656 B/op	      56 allocs/op
// Benchmark_Sign/sign_ecdsa-P256_4k
// Benchmark_Sign/sign_ecdsa-P256_4k-104       	   10752	    114306 ns/op	    2719 B/op	      37 allocs/op
// Benchmark_Sign/sign_ecdsa-P384_4k
// Benchmark_Sign/sign_ecdsa-P384_4k-104       	     253	   4663023 ns/op	    2920 B/op	      38 allocs/op
// Benchmark_Sign/sign_ed25519_4k
// Benchmark_Sign/sign_ed25519_4k-104				2299	    521442 ns/op	      64 B/op	       1 allocs/op
// Benchmark_Sign/sign_schnorr-ed25519_4k
// Benchmark_Sign/sign_schnorr-ed25519_4k-104  	     493	   2252567 ns/op	    3822 B/op	      48 allocs/op
// PASS
// coverage: 1.7% of statements
// ok  	github.com/Laisky/go-utils/v4/crypto	16.896s
func Benchmark_Sign(b *testing.B) {
	raw4k, err := Salt(4 * 1024)
	require.NoError(b, err)

	b.Run("sign rsa-2048 4k", func(b *testing.B) {
		prikey, err := NewRSAPrikey(RSAPrikeyBits2048)
		require.NoError(b, err)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := SignByRSAWithSHA256(prikey, raw4k)
			require.NoError(b, err)
		}
	})

	b.Run("sign rsa-4096 4k", func(b *testing.B) {
		prikey, err := NewRSAPrikey(RSAPrikeyBits4096)
		require.NoError(b, err)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := SignByRSAWithSHA256(prikey, raw4k)
			require.NoError(b, err)
		}
	})

	b.Run("sign ecdsa-P256 4k", func(b *testing.B) {
		prikey, err := NewECDSAPrikey(ECDSACurveP256)
		require.NoError(b, err)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := SignByECDSAWithSHA256(prikey, raw4k)
			require.NoError(b, err)
		}
	})

	b.Run("sign ecdsa-P384 4k", func(b *testing.B) {
		prikey, err := NewECDSAPrikey(ECDSACurveP384)
		require.NoError(b, err)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := SignByECDSAWithSHA256(prikey, raw4k)
			require.NoError(b, err)
		}
	})

	b.Run("sign ed25519 4k", func(b *testing.B) {
		prikey, err := NewEd25519Prikey()
		require.NoError(b, err)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := prikey.Sign(rand.Reader, raw4k, crypto.Hash(0))
			require.NoError(b, err)
		}
	})

	b.Run("sign schnorr-ed25519 4k", func(b *testing.B) {
		suite := edwards25519.NewBlakeSHA256Ed25519()
		keyPair := dediskey.NewKeyPair(suite)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := SignBySchnorrSha256(suite, keyPair.Private, bytes.NewReader(raw4k))
			require.NoError(b, err)
		}
	})

}

func TestVerifyByEd25519(t *testing.T) {
	t.Parallel()

	prikey, err := NewEd25519Prikey()
	require.NoError(t, err)
	pubkey := prikey.Public().(ed25519.PublicKey)

	content := []byte("hello, world")

	sig, err := SignByEd25519WithSHA512(prikey, bytes.NewReader(content))
	require.NoError(t, err)

	err = VerifyByEd25519WithSHA512(pubkey, bytes.NewReader(content), sig)
	require.NoError(t, err)

	t.Run("invalid sig", func(t *testing.T) {
		err := VerifyByEd25519WithSHA512(pubkey, bytes.NewReader(content), []byte("2l3fj238f83"))
		require.ErrorContains(t, err, "invalid signature")
	})

	t.Run("invalid key", func(t *testing.T) {
		prikey, err := NewEd25519Prikey()
		require.NoError(t, err)
		pubkey := prikey.Public().(ed25519.PublicKey)

		err = VerifyByEd25519WithSHA512(pubkey, bytes.NewReader(content), []byte("2l3fj238f83"))
		require.ErrorContains(t, err, "invalid signature")
	})
}

func TestHMAC(t *testing.T) {
	t.Parallel()

	for _, keyLen := range []int{
		1, 1024, 10240,
	} {
		keyLen := keyLen
		key, err := Salt(keyLen)
		require.NoError(t, err)

		t.Run(fmt.Sprintf("keyLen=%d", keyLen), func(t *testing.T) {
			t.Parallel()

			for _, plainLen := range []int{
				1, 1024, 10240,
			} {
				plainLen := plainLen
				plain, err := Salt(plainLen)
				require.NoError(t, err)

				t.Run(fmt.Sprintf("plainLen=%d", plainLen), func(t *testing.T) {
					t.Parallel()

					hmac1, err := HMACSha256(key, bytes.NewReader(plain))
					require.NoError(t, err)

					hmac2, err := HMACSha256(key, bytes.NewReader(plain))
					require.NoError(t, err)
					require.Equal(t, hmac1, hmac2)

					t.Run("incorrect plain", func(t *testing.T) {
						newplain, err := Salt(plainLen)
						require.NoError(t, err)

						hmacIncorrect, err := HMACSha256(key, bytes.NewReader(newplain))
						require.NoError(t, err)
						require.NotEqual(t, hmac1, hmacIncorrect)
					})

					t.Run("incorrect key", func(t *testing.T) {
						newkey, err := Salt(keyLen)
						require.NoError(t, err)

						hmacIncorrect, err := HMACSha256(newkey, bytes.NewReader(plain))
						require.NoError(t, err)
						require.NotEqual(t, hmac1, hmacIncorrect)
					})
				})
			}
		})
	}
}
