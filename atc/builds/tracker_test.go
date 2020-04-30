package builds_test

import (
	"time"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	"github.com/concourse/concourse/atc/builds"
	"github.com/concourse/concourse/atc/db"
	"github.com/concourse/concourse/atc/db/dbfakes"
	"github.com/concourse/concourse/atc/engine"
	"github.com/concourse/concourse/atc/engine/enginefakes"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TrackerSuite struct {
	suite.Suite
	*require.Assertions

	fakeBuildFactory *dbfakes.FakeBuildFactory
	fakeEngine       *enginefakes.FakeEngine

	tracker *builds.Tracker
}

func (s *TrackerSuite) SetupTest() {
	s.fakeBuildFactory = new(dbfakes.FakeBuildFactory)
	s.fakeEngine = new(enginefakes.FakeEngine)

	s.tracker = builds.NewTracker(
		lagertest.NewTestLogger("test"),
		s.fakeBuildFactory,
		s.fakeEngine,
	)
}

func (s *TrackerSuite) TestTrackRunsStartedBuilds() {
	returnedBuilds := []db.Build{}

	for i := 0; i < 3; i++ {
		fakeBuild := new(dbfakes.FakeBuild)
		fakeBuild.IDReturns(i + 1)

		returnedBuilds = append(returnedBuilds, fakeBuild)
	}

	s.fakeBuildFactory.GetAllStartedBuildsReturns(returnedBuilds, nil)

	running := make(chan db.Build, 3)
	s.fakeEngine.NewBuildStub = func(build db.Build) engine.Runnable {
		engineBuild := new(enginefakes.FakeRunnable)
		engineBuild.RunStub = func(lager.Logger) {
			running <- build
		}

		return engineBuild
	}

	err := s.tracker.Track()
	s.NoError(err)

	ranBuilds := []db.Build{
		<-running,
		<-running,
		<-running,
	}
	s.ElementsMatch(ranBuilds, returnedBuilds)
}

func (s *TrackerSuite) TestTrackDoesntTrackAlreadyRunningBuilds() {
	fakeBuild := new(dbfakes.FakeBuild)
	fakeBuild.IDReturns(1)
	s.fakeBuildFactory.GetAllStartedBuildsReturns([]db.Build{fakeBuild}, nil)

	running := make(chan db.Build, 3)
	wait := make(chan struct{})
	s.fakeEngine.NewBuildStub = func(build db.Build) engine.Runnable {
		engineBuild := new(enginefakes.FakeRunnable)
		engineBuild.RunStub = func(lager.Logger) {
			running <- build
			<-wait
		}

		return engineBuild
	}

	err := s.tracker.Track()
	s.NoError(err)

	// wait for the build to be running
	s.Eventually(s.tracker.Running, time.Second, time.Millisecond)

	// try to run the build again
	err = s.tracker.Track()
	s.NoError(err)

	// allow the build to "finish"
	close(wait)
	s.Eventually(func() bool { return !s.tracker.Running() }, time.Second, time.Millisecond)

	// confirm that only one build ran
	s.Equal(<-running, fakeBuild)
	s.Empty(running)
}

func (s *TrackerSuite) TestReleaseReleasesEngine() {
	s.tracker.Release()
	s.Equal(s.fakeEngine.ReleaseAllCallCount(), 1)
}
