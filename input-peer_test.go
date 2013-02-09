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
