package core
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

// The largest 63-bit prime number (60 rounds for equality)
// const LargePrime int64 = 9223372036854775783 

// (13 bits)
// const LargePrime int64 = 571710324769

// (11 bits)
//const LargePrime int64 = 571710243073 

// (2 bits)
const LargePrime int64 = 8858370049 

// Distribute a secret among a certain number of peers. Use the large Prime number above
// as the field
func DistributeSecret (num int64, shares int32) ([]int64) {
    return ShamirSecretSharing (num, shares, LargePrime)
}

// Interpolate and reconstruct the secret in the prime field specified by LargePrime
func ReconstructSecret (shares *[]int64, sharesAvailable *[]bool, nshare int32) (int64) {
   return Interpolate (shares, sharesAvailable, nshare, LargePrime)
}

// The actual secret sharing function, computes coefficients for the polynomial
func ShamirSecretSharing (num int64, shares int32, prime int64) ([]int64){
    if shares <= 0 {
        return nil
    }
    
    if prime <= 2 {
        return nil
    }
    degree := (shares - 1) / 2 // Degree must be no greater than (shares - 1)/2 to allow for multiplication 
    coeffs := GenerateCoefficients (degree, big.NewInt(num), prime) 
    primebig := big.NewInt(prime)
    return SharesForCoefficients (shares, &coeffs, degree, primebig)
}

// Generate coefficients
func GenerateCoefficients (degree int32, c *big.Int, prime int64) ([]big.Int) {
    coeffs := make([]big.Int, degree + 1)
    coeffs[0].Set(c) // The constant is the constant coefficient for the argument
    for coeff := int32(1); coeff < degree + 1; coeff++ {
      var tmp int64
      binary.Read(rand.Reader, binary.LittleEndian, &tmp)
      coeffs[coeff].Set(big.NewInt(tmp % prime))
    }
    return coeffs
}

// Compute shares given a polynomial (specified as a set of coefficients
func SharesForCoefficients (shares int32, coeffs *[]big.Int, degree int32, primebig *big.Int) ([]int64) {
    result := make([]int64, shares)
    for share := int32(0); share < shares; share++ {
        alpha := int64(share + 2) // TODO: The Sepia code seems to indicate that an alpha of 0 or 1 is in fact untenable, 
                           // 0 makes sense, 1 does not?
        tmp := big.NewInt(0)
        for coeff := int32(0); coeff < degree + 1; coeff++ {
          fexp := FastExp(alpha, big.NewInt(int64(coeff)), primebig)
          mul := big.NewInt(1)
          mul.Mul(&(*coeffs)[coeff], fexp)
          sum := big.NewInt(0)
          sum.Add (tmp, mul)
          tmp.Mod(sum, primebig)
        }
        result[share] = tmp.Int64()
    }
    return result
}

// Fast exponentiation
func FastExp (n int64, exp *big.Int, field *big.Int) (*big.Int) {
    nBig := big.NewInt(int64(n))
    result := big.NewInt(0)
    return result.Exp(nBig, exp, field)
}

// Lagrangian interpolation to reconstruct the constant factor for a polynomial
func Interpolate (shares *[]int64, shareAvailable *[]bool, nshare int32, prime int64) (int64) {
    primebig := big.NewInt(prime)
    resultbig := big.NewInt(0)
    for share := int32(0); share < nshare; share++ {
      if ((*shareAvailable)[share]) {
        numerator := big.NewInt(1)
        denominator := big.NewInt(1)
        for otherShare := int32(0); otherShare < nshare; otherShare++ {
          if (otherShare == share) || (!(*shareAvailable)[otherShare]) {
            continue
          }
          numerator.Mul(numerator, big.NewInt(-1 * int64(otherShare + 2)))
          denominator.Mul(denominator, big.NewInt(int64(share - otherShare)))
        }
        lagrange := big.NewInt(1)
        lagrange.DivMod(numerator, denominator, primebig)
        primebig.Set(big.NewInt(prime)) // Strangely divmod sets primebig to 0, divmod bad
        tmp := big.NewInt(1)
        tmp.Mul(big.NewInt((*shares)[share]), lagrange)
        resultTmp := big.NewInt(0)
        resultTmp.Add(tmp, resultbig)
        resultbig.Mod(resultTmp, primebig)
      }
    }
    return resultbig.Int64()
}
