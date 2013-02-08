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
   "fmt"
)

const largePrime int64 = 9223372036854775783
func DistributeSecret (num int64, shares int32) ([]int64) {
    return ShamirSecretSharing (num, shares, largePrime)
}

func ReconstructSecret (shares *[]int64, sharesAvailable *[]bool, nshare int32) (int64) {
   return Interpolate (shares, sharesAvailable, nshare, largePrime)
}


func ShamirSecretSharing (num int64, shares int32, prime int64) ([]int64){
    if shares <= 0 {
        return nil
    }
    
    if prime <= 2 {
        return nil
    }
    degree := (shares - 1) / 2 // Degree must be no greater than (shares - 1)/2 to allow for multiplication 
    fmt.Printf("Shares = %d, degree = %d\n", shares, degree)
    coeffs := make([]big.Int, degree + 1)
    coeffs[0].Set(big.NewInt(num)) // The constant is the constant coefficient for the argument
    for coeff := int32(1); coeff < degree + 1; coeff++ {
      var tmp int64
      binary.Read(rand.Reader, binary.LittleEndian, &tmp)
      coeffs[coeff].Set(big.NewInt(tmp % prime))
    }
    primebig := big.NewInt(prime)
    return SharesForCoefficients (shares, &coeffs, degree, primebig)
}

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

func FastExp (n int64, exp *big.Int, field *big.Int) (*big.Int) {
    nBig := big.NewInt(int64(n))
    result := big.NewInt(0)
    return result.Exp(nBig, exp, field)
}

func Interpolate (shares *[]int64, shareAvailable *[]bool, nshare int32, prime int64) (int64) {
    primebig := big.NewInt(prime)
    fmt.Printf("Prime = %d, Prime Big = %d\n", prime, primebig.Int64())
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
        fmt.Printf("Numerator = %d, Denominator = %d\n", numerator, denominator)
        fmt.Printf("Prime = %d, Prime Big = %d\n", prime, primebig.Int64())
        lagrange.DivMod(numerator, denominator, primebig)
        primebig.Set(big.NewInt(prime)) // Strangely divmod sets primebig to 0, divmod bad
        fmt.Printf("Prime = %d, Prime Big = %d, Lagrange = %d, Shares = %d\n", prime, primebig.Int64(), lagrange.Int64(), (*shares)[share])
        tmp := big.NewInt(1)
        tmp.Mul(big.NewInt((*shares)[share]), lagrange)
        resultTmp := big.NewInt(0)
        resultTmp.Add(tmp, resultbig)
        fmt.Printf("%d Prime Big = %d\n", resultTmp.Int64(), primebig.Int64())
        resultbig.Mod(resultTmp, primebig)
      }
    }
    return resultbig.Int64()
}
