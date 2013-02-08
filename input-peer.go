package smpc
/* An SMPC input peer in Go, mostly written as a way to
   - Teach myself Go
   - Figure out what the hell is going on with SMPC
*/
//import "math"
import (
   "encoding/binary"
   "crypto/rand"
   "math/big"
)

const largePrime uint64 = 9223372036854775783

func ShamirSecretSharing (num int64, shares int32, prime int64) ([]int64){
    if shares <= 0 {
        return nil
    }
    
    if prime <= 2 {
        return nil
    }
    result := make([]int64, 0, shares)
    degree := (shares - 1) / 2 // Degree must be no greater than (shares - 1)/2 to allow for multiplication 
    coeffs := make([]big.Int, 0, degree + 1)
    coeffs[0].Set(big.NewInt(num)) // The constant is the constant coefficient for the argument
    for coeff := int32(1); coeff < degree + 1; coeff++ {
      var tmp int64
      binary.Read(rand.Reader, binary.LittleEndian, &tmp)
      coeffs[coeff].Set(big.NewInt(tmp % prime))
    }
    primebig := big.NewInt(prime)
    for share := int32(0); share < shares; share++ {
        alpha := int64(share + 2) // TODO: The Sepia code seems to indicate that an alpha of 0 or 1 is in fact untenable, 
                           // 0 makes sense, 1 does not?
        tmp := big.NewInt(0)
        for coeff := int32(0); coeff < degree + 1; coeff++ {
          tmp.Mod(tmp.Add(tmp, fastExp(alpha, &coeffs[coeff], primebig)) ,primebig)
        }
        result[share] = tmp.Int64()
    }
    return result
}

func fastExp (n int64, exp *big.Int, field *big.Int) (*big.Int) {
    nBig := big.NewInt(int64(n))
    result := big.NewInt(0)
    return result.Exp(nBig, exp, field)
}

func interpolate (shares []int64, shareAvailable []bool, nshare int32, prime int64) (int64) {
    lagrangeWeights := make([]int64, 0, nshare)
    result := int64(0)
    primebig := big.NewInt(prime)
    for share := int32(0); share < nshare; share++ {
      if (!shareAvailable[share]) {
        lagrangeWeights[share] = 0
      } else {
        numerator := big.NewInt(1)
        denominator := big.NewInt(1)
        for otherShare := int32(0); otherShare < nshare; otherShare++ {
          if (otherShare == share) || (!shareAvailable[otherShare]) {
            continue
          }
          numerator.Mul(numerator, big.NewInt(-1 * shares[otherShare]))
          denominator.Mul(denominator, big.NewInt(shares[share] - shares[otherShare]))
        }
        lagrange := big.NewInt(1)
        lagrange.DivMod(numerator, denominator, primebig)
        lagrangeWeights[share] = lagrange.Int64()
      }
      
    }
    return result
}
