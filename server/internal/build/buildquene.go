package build

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/codenames"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/docker/docker/client"
)

func init() {
	NewBuildQueueManager(1)
}

// BuildTask defines a task structure for the build process
type BuildTask struct {
	req     *clientpb.Generate      // The build request
	result  chan *clientpb.Artifact // Channel to send back the build result
	err     chan error              // Channel to send back error in case of failure
	builder *models.Builder
}

// BuildQueueManager manages the build task queue
type BuildQueueManager struct {
	taskQueue   chan BuildTask // Channel holding tasks in the queue
	workerCount int            // Number of worker goroutines
	wg          sync.WaitGroup // WaitGroup to wait for all workers to finish
}

// GlobalBuildQueueManager is the global build queue manager instance
var GlobalBuildQueueManager *BuildQueueManager
var queneOnce sync.Once

// NewBuildQueueManager creates a new BuildQueueManager instance
func NewBuildQueueManager(workerCount int) *BuildQueueManager {
	queneOnce.Do(func() {
		GlobalBuildQueueManager = &BuildQueueManager{
			taskQueue:   make(chan BuildTask, 100), // Buffer size of 100 tasks
			workerCount: workerCount,
		}
		GlobalBuildQueueManager.Start()
	})
	return GlobalBuildQueueManager
}

// Start starts the worker goroutines that will process the tasks
func (bqm *BuildQueueManager) Start() {
	for i := 0; i < bqm.workerCount; i++ {
		bqm.wg.Add(1)
		go bqm.worker(i)
	}
}

// Stop stops the queue manager and waits for all workers to finish
func (bqm *BuildQueueManager) Stop() {
	close(bqm.taskQueue) // Close the task queue to signal workers to stop
	bqm.wg.Wait()        // Wait for all workers to finish
}

// worker processes the tasks from the queue
func (bqm *BuildQueueManager) worker(id int) {
	defer bqm.wg.Done()               // Ensure to mark the worker as done when finished
	for task := range bqm.taskQueue { // Continuously fetch tasks from the queue
		// Execute the build task and send the result or error back
		result, err := bqm.executeBuild(task.req, task.builder)
		if err != nil {
			task.err <- err // Send error if build fails
		} else {
			task.result <- result // Send the result if build succeeds
		}
	}
}

// executeBuild executes the build process based on the request
func (bqm *BuildQueueManager) executeBuild(req *clientpb.Generate, builder *models.Builder) (*clientpb.Artifact, error) {
	target, ok := consts.GetBuildTarget(req.Target)
	if !ok {
		return nil, errs.ErrInvalidateTarget
	}

	// Ensure valid build target
	if req.Type == consts.CommandBuildPulse && !strings.Contains(target.Name, "windows") {
		return nil, errs.ErrInvalidateTarget
	}

	// Prepare build request
	req.Name = builder.Name
	profileByte, err := GenerateProfile(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create config: %v", err)
	}

	// Set feature if empty
	if req.Feature == "" {
		profile, _ := types.LoadProfile([]byte(profileByte))
		req.Feature = strings.Join(profile.Implant.Modules, ",")
	}

	// Get Docker client
	cli, err := GetDockerClient()
	if err != nil {
		return nil, err
	}

	logs.Log.Infof("start to build %s ...", req.Target)

	// Handle different build types
	switch req.Type {
	case consts.CommandBuildBeacon:
		err = BuildBeacon(cli, req)
	case consts.CommandBuildBind:
		err = BuildBind(cli, req)
	case consts.CommandBuildPrelude:
		err = BuildPrelude(cli, req)
	case consts.CommandBuildModules:
		err = BuildModules(cli, req)
	case consts.CommandBuildPulse:
		err = bqm.handleBuildPulse(cli, req, target)
	default:
		err = fmt.Errorf("unknown build type: %s", req.Type)
	}

	if err != nil {
		return nil, err
	}

	// Finalize build and move output
	return bqm.finalizeBuild(req, builder, target)
}

// handleBuildPulse handles the specific logic for Pulse builds
func (bqm *BuildQueueManager) handleBuildPulse(cli *client.Client, req *clientpb.Generate, target *consts.BuildTarget) error {
	var artifactID uint32
	if req.ArtifactId != 0 {
		artifactID = req.ArtifactId
	} else {
		profile, _ := db.GetProfile(req.ProfileName)
		yamlID := profile.Pulse.Extras["flags"].(map[string]interface{})["artifact_id"].(int)
		if uint32(yamlID) != 0 {
			artifactID = uint32(yamlID)
		} else {
			artifactID = 0
		}
	}

	idBuilder, err := db.GetArtifactById(artifactID)
	if err != nil && !errors.Is(err, db.ErrRecordNotFound) {
		return fmt.Errorf("getting artifact error: %v", err)
	}

	// Handle case when artifact is not found
	if errors.Is(err, db.ErrRecordNotFound) {
		logs.Log.Infof("artifact not found, creating new Beacon build")
		return bqm.handleNewBeaconBuild(cli, req, artifactID, target)
	}
	if !idBuilder.IsSRDI {
		idBuilder.IsSRDI = true
		_, err := SRDIArtifact(idBuilder, target.OS, target.Arch)
		if err != nil {
			return err
		}
	}

	return BuildPulse(cli, req)
}

// handleNewBeaconBuild handles the creation of a new Beacon build
func (bqm *BuildQueueManager) handleNewBeaconBuild(cli *client.Client, req *clientpb.Generate, artifactID uint32, target *consts.BuildTarget) error {
	beaconReq := &clientpb.Generate{
		Name:        codenames.GetCodename(),
		ProfileName: req.ProfileName,
		Address:     req.Address,
		Type:        consts.CommandBuildBeacon,
		Target:      req.Target,
		Modules:     req.Modules,
		Ca:          req.Ca,
		Params:      req.Params,
		Srdi:        true,
	}

	// Set the correct target for Windows
	if beaconReq.Target == consts.TargetX86Windows {
		beaconReq.Target = consts.TargetX86WindowsGnu
	} else {
		beaconReq.Target = consts.TargetX64WindowsGnu
	}

	var beaconBuilder *models.Builder
	var err error
	if artifactID != 0 {
		beaconBuilder, err = db.SaveArtifactFromID(beaconReq, artifactID, consts.ArtifactFromDocker)
	} else {
		beaconBuilder, err = db.SaveArtifactFromGenerate(beaconReq)
	}
	if err != nil {
		logs.Log.Errorf("error saving artifact: %v", err)
		return err
	}

	// Add Beacon build task to queue asynchronously
	go func() {
		_, err := GlobalBuildQueueManager.AddTask(beaconReq, beaconBuilder)
		if err != nil {
			logs.Log.Errorf("Error adding BuildBeacon task: %v", err)
		}
	}()

	// Generate profile and build Pulse
	req.ArtifactId = beaconBuilder.ID
	_, err = GenerateProfile(req)
	if err != nil {
		return fmt.Errorf("failed to create config: %v", err)
	}
	return BuildPulse(cli, req)
}

// finalizeBuild moves the build output and updates the builder path
func (bqm *BuildQueueManager) finalizeBuild(req *clientpb.Generate, builder *models.Builder, target *consts.BuildTarget) (*clientpb.Artifact, error) {
	_, artifactPath, err := MoveBuildOutput(req.Target, req.Type)
	if err != nil {
		logs.Log.Errorf("move build output error: %v", err)
		return nil, err
	}

	absArtifactPath, err := filepath.Abs(artifactPath)
	if err != nil {
		return nil, err
	}

	builder.Path = absArtifactPath
	err = db.UpdateBuilderPath(builder)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(absArtifactPath)
	if err != nil {
		return nil, err
	}
	//_, err = SRDIArtifact(&builder, target.OS, target.Arch)
	//if err != nil {
	//	return nil, err
	//}
	if builder.IsSRDI {

		if builder.Type == consts.CommandBuildPulse {
			logs.Log.Infof("objcopy start ...")
			_, err = OBJCOPYPulse(builder, target.OS, target.Arch)
			if err != nil {
				return nil, fmt.Errorf("objcopy error: %v", err)
			}
			logs.Log.Infof("objcopy end ...")
		} else {
			_, err = SRDIArtifact(builder, target.OS, target.Arch)
			if err != nil {
				return nil, err
			}
		}
	}

	return builder.ToArtifact(data), nil
}

// AddTask adds a build task to the queue and waits for the result
func (bqm *BuildQueueManager) AddTask(req *clientpb.Generate, builder *models.Builder) (*clientpb.Artifact, error) {
	resultChan := make(chan *clientpb.Artifact) // Channel to receive the result
	errChan := make(chan error)                 // Channel to receive errors
	task := BuildTask{
		req:     req,
		result:  resultChan,
		err:     errChan,
		builder: builder,
	}
	bqm.taskQueue <- task // Add the task to the queue

	// Wait for either a result or an error
	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errChan:
		return nil, err
	}
}
