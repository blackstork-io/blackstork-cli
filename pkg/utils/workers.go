package utils

import (
	"context"
	"log/slog"
)

func worker[inputT any, outputT any](
	ctx context.Context,
	log *slog.Logger,
	id int,
	inputCh <-chan inputT,
	outputCh chan<- outputT,
	errCh chan<- error,
	workFunc func(inputT) (outputT, error),
) {
	log.DebugContext(ctx, "Starting a worker", "id", id)
	counter := 0
	for input := range inputCh {
		output, err := workFunc(input)

		if err != nil {
			log.ErrorContext(ctx, "Error processing an input", "err", err)
			errCh <- err
			continue
		}

		errCh <- nil
		outputCh <- output
		counter += 1
	}
	log.DebugContext(ctx, "Worker finished", "id", id, "inputs_processed", counter)
}

func RunWorkers[inputT any, outputT any](
	ctx context.Context,
	log *slog.Logger,
	inputs []inputT,
	workerCount int,
	workerFunc func(inputT) (outputT, error),
) ([]outputT, []error) {
	inputsCount := len(inputs)

	inputCh := make(chan inputT, inputsCount)
	outputCh := make(chan outputT, inputsCount)
	errCh := make(chan error)

	for w := 1; w <= workerCount; w++ {
		go worker(ctx, log, w, inputCh, outputCh, errCh, workerFunc)
	}

	for j := range inputsCount {
		inputCh <- inputs[j]
	}
	close(inputCh)

	errors := []error{}
	for range inputsCount {
		if err := <-errCh; err != nil {
			errors = append(errors, err)
		}
	}

	outputs := []outputT{}
	for range (inputsCount - len(errors)) {
		output := <-outputCh
		outputs = append(outputs, output)
	}

	return outputs, errors
}
