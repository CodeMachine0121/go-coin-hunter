package utilities

import (
	"strconv"

	"github.com/CodeMachine0121/go-coin-hunter/internal/domain/models/domains"
	"github.com/gin-gonic/gin"
)

// PipelineRunIDFrom reads the :pipelineRunId route parameter shared by every run-scoped route; anything but a positive
// integer is domains.ErrInvalidPipelineRunID.
func PipelineRunIDFrom(context *gin.Context) (uint, error) {
	pipelineRunID, parseError := strconv.ParseUint(context.Param("pipelineRunId"), 10, 64)
	if parseError != nil || pipelineRunID == 0 {
		return 0, domains.ErrInvalidPipelineRunID
	}

	return uint(pipelineRunID), nil
}
