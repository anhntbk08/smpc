package smpc
/* An SMPC input peer in Go, mostly written as a way to
   - Teach myself Go
   - Figure out what the hell is going on with SMPC
*/
import (
      "testing"
      . "launchpad.net/gocheck"
      "math/big"
      "fmt"
)

func Test (t *testing.T) {TestingT(t) }

type InputPeerSuite struct{}
var _ = Suite(&InputPeerSuite{})

func (s *InputPeerSuite) TestInputPeer_1(c *C) {
    manualCoeffs := []big.Int{*big.NewInt(6), *big.NewInt(5)}
    primeBig := big.NewInt(10091)

    calcShares := SharesForCoefficients (3, &manualCoeffs, 1, primeBig)

    c.Assert (len(calcShares), Equals, 3)
    c.Assert (calcShares[0], Equals, int64(16))
    c.Assert (calcShares[1], Equals, int64(21))
    c.Assert (calcShares[2], Equals, int64(26))

    shares := DistributeSecret (int64(3231524), 3)
    c.Assert(len(shares), Equals, 3) 

    fetest := FastExp(2,big.NewInt(3), big.NewInt(7))
    c.Check(fetest.Int64(), Equals, int64(1))
    fetest = FastExp(2,big.NewInt(0), big.NewInt(7))
    c.Check(fetest.Int64(), Equals, int64(1))
    reconstructed := ReconstructSecret (&shares, &[]bool{true, true, true}, 3) 
    c.Check(reconstructed, Equals, int64(3231524))
    reconstructed = ReconstructSecret (&shares, &[]bool{true, true, false}, 3)
    c.Check(reconstructed, Equals, int64(3231524))
    reconstructed = ReconstructSecret (&shares, &[]bool{true, false, false}, 3)
    c.Check(reconstructed, Not(Equals), int64(3231524))
}

func (s *InputPeerSuite) TestAddSub_1(c *C) {
   shares1 := DistributeSecret (int64(1), 3)
   shares2 := DistributeSecret (int64(1), 3)
   shares3 := make([]int64, 3)
   for i := 0; i < len(shares1); i++ {
       fmt.Printf("%d %d\n", shares1[i], shares2[i])
       shares3[i] = Add(shares1[i], shares2[i])
   }
   reconstructed := ReconstructSecret(&shares3, &[]bool{true, false, true}, 3)
   c.Check(reconstructed, Equals, int64(2))
}

func Multiply (a int64, b int64) (int64) {
   shares1 := DistributeSecret (a, 3)
   shares2 := DistributeSecret (b, 3)
   mshares := make([][]int64, 3)
   for i := 0; i < len(mshares); i++ {
     mshares[i] = make([]int64, 3)
   }
   rshares := make([]int64, 3)
   for i := 0; i < len(shares1); i++ {
      temp := MultShares (shares1[i], shares2[i], 3)
      for j := 0; j < 3; j++ {
        mshares[j][i] = temp[j]
      }
   }
   for i := 0; i < len(mshares); i++ {
     rshares[i] = ReconstructSecret(&mshares[i], &[]bool{true, true, true}, 3)
   }
   return ReconstructSecret(&rshares, &[]bool{true, true, true}, 3)
}

func (s *InputPeerSuite) TestMul_1(c *C) {
   c.Check(Multiply(int64(1), int64(1)), Equals, int64(1))
   c.Check(Multiply(int64(7), int64(1)), Equals, int64(7))
   c.Check(Multiply(int64(2), int64(4)), Equals, int64(8))
   c.Check(Multiply(int64(15), int64(13)), Equals, int64(195))
   c.Check(Multiply(int64(169), int64(169)), Equals, int64(28561))
   c.Check(Multiply(int64(12), int64(24)), Equals, int64(288))
   c.Check(Multiply(int64(2), int64(2)), Equals, int64(4))
}

func (s *InputPeerSuite) TestMul_2 (c *C) {
   shares1 := DistributeSecret (int64(7), 3)
   shares2 := DistributeSecret (int64(8), 3)
   mshares := make([][]int64, 3)
   for i := 0; i < len(mshares); i++ {
     mshares[i] = make([]int64, 3)
   }
   rshares0 := make([]int64, 3)
   for i := 0; i < len(shares1); i++ {
      temp := MultShares (shares1[i], shares2[i], 3)
      for j := 0; j < 3; j++ {
        mshares[j][i] = temp[j]
      }
   }
   for i := 0; i < len(mshares); i++ {
     rshares0[i] = ReconstructSecret(&mshares[i], &[]bool{true, true, true}, 3)
   }
   shares1 = DistributeSecret (int64(7), 3)
   shares2 = DistributeSecret (int64(8), 3)
   mshares = make([][]int64, 3)
   for i := 0; i < len(mshares); i++ {
     mshares[i] = make([]int64, 3)
   }
   rshares1 := make([]int64, 3)
   for i := 0; i < len(shares1); i++ {
      temp := MultShares (shares1[i], shares2[i], 3)
      for j := 0; j < 3; j++ {
        mshares[j][i] = temp[j]
      }
   }
   for i := 0; i < len(mshares); i++ {
     rshares1[i] = ReconstructSecret(&mshares[i], &[]bool{true, true, true}, 3)
   }

   for i := 0; i < len(rshares0); i++ {
       c.Check(rshares0[i], Not(Equals), rshares1[i])
   }
}
