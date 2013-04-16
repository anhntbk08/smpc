package core
/* 
  Multiplication for SMPC
  Multiplication proceeds in two stages:
  1. Everyone computes a product of shares. While this is a polynoial with the right constant, unfortunately it
     has higher degree, and cannot hence be interpolated. To get over this, everyone also computes shares for their
     secret using a degree nshares - 1 polynomial. Send one of these shares to each of the peers
  2. Everyone takes the set of n shares, and interpolates, generating a point on a lower degree polynomial 
*/
import (
        "math/big"
        )

func MultShares (a int64, b int64, nshares int32) ([]int64) {
   return SmpcMultShares(a, b, nshares, LargePrime)
}

func MultUnderModPrime (a int64, b int64, prime int64) (int64) {
   aBig := big.NewInt(a)
   bBig := big.NewInt(b)
   primeBig := big.NewInt(prime)
   prod := big.NewInt(1)
   prod.Mul(aBig, bBig)
   prod.Mod(prod, primeBig) // Product of share is a value of an nshare polynomial
   return prod.Int64()
}

func MultUnderMod (a int64, b int64) (int64) {
    return MultUnderModPrime(a, b, LargePrime)
}

func SmpcMultShares (a int64, b int64, nshares int32, prime int64) ([]int64) {
   shares := DistributeSecret(MultUnderModPrime (a, b, prime), nshares)
   return shares
}

func MultCombineShares (shares *[]int64, nshares int32) (int64) {
   return SmpcMultCombineShares(shares, nshares, LargePrime)
}

func SmpcMultCombineShares (shares *[]int64, nshares int32, prime int64) (int64) {
  hasShare := make([]bool, nshares)
  for i := int32(0); i < nshares; i++ {
    hasShare[i] = true
  }
  return Interpolate(shares, &hasShare, nshares, prime)   
}
