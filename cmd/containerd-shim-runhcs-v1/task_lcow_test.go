package main

import (
	"context"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/containerd/containerd/errdefs"
)

func setupTestLcowTask(t *testing.T) (*lcowTask, *testShimExec, *testShimExec) {
	initExec := &testShimExec{
		id:  "", // init exec
		pid: int(rand.Int31()),
	}
	lt := &lcowTask{
		id:   t.Name(),
		init: initExec,
	}
	secondExec := &testShimExec{
		id:  strconv.Itoa(rand.Int()),
		pid: int(rand.Int31()),
	}
	lt.execs.Store(secondExec.id, secondExec)
	return lt, initExec, secondExec
}

func Test_lcowTask_ID(t *testing.T) {
	lt, _, _ := setupTestLcowTask(t)

	if lt.ID() != t.Name() {
		t.Fatalf("expect ID: '%s', got: '%s'", t.Name(), lt.ID())
	}
}

func Test_lcowTask_GetExec_Empty_Success(t *testing.T) {
	lt, i, _ := setupTestLcowTask(t)

	e, err := lt.GetExec("")
	if err != nil {
		t.Fatalf("should not have failed with error: %v", err)
	}
	if i != e {
		t.Fatal("should of returned the init exec on empty")
	}
}

func Test_lcowTask_GetExec_UnknownExecID_Error(t *testing.T) {
	lt, _, _ := setupTestLcowTask(t)

	e, err := lt.GetExec("shouldnotmatch")

	verifyExpectedError(t, e, err, errdefs.ErrNotFound)
}

func Test_lcowTask_GetExec_2ndID_Success(t *testing.T) {
	lt, _, second := setupTestLcowTask(t)

	e, err := lt.GetExec(second.id)
	if err != nil {
		t.Fatalf("should not have failed with error: %v", err)
	}
	if second != e {
		t.Fatal("should of returned the second exec")
	}
}

func Test_lcowTask_KillExec_UnknownExecID_Error(t *testing.T) {
	lt, _, _ := setupTestLcowTask(t)

	err := lt.KillExec(context.TODO(), "thisshouldnotmatch", 0xf, false)

	verifyExpectedError(t, nil, err, errdefs.ErrNotFound)
}

func Test_lcowTask_KillExec_InitExecID_Unexited2ndExec_Error(t *testing.T) {
	lt, _, _ := setupTestLcowTask(t)

	err := lt.KillExec(context.TODO(), "", 0xf, false)

	verifyExpectedError(t, nil, err, errdefs.ErrFailedPrecondition)
}

func Test_lcowTask_KillExec_InitExecID_All_Success(t *testing.T) {
	lt, init, second := setupTestLcowTask(t)

	err := lt.KillExec(context.TODO(), "", 0xf, true)
	if err != nil {
		t.Fatalf("should not have failed, got: %v", err)
	}
	if init.state != shimExecStateExited {
		t.Fatalf("init should be in exited state got: %v", init.state)
	}
	if second.state != shimExecStateExited {
		t.Fatalf("2nd exec should be in exited state got: %v", second.state)
	}
}

func Test_lcowTask_KillExec_2ndExecID_Success(t *testing.T) {
	lt, _, second := setupTestLcowTask(t)

	err := lt.KillExec(context.TODO(), second.id, 0xf, false)
	if err != nil {
		t.Fatalf("should not have failed, got: %v", err)
	}
	if second.state != shimExecStateExited {
		t.Fatalf("2nd exec should be in exited state got: %v", second.state)
	}
}

func Test_lcowTask_KillExec_2ndExecID_All_Error(t *testing.T) {
	lt, _, second := setupTestLcowTask(t)

	err := lt.KillExec(context.TODO(), second.id, 0xf, true)

	verifyExpectedError(t, nil, err, errdefs.ErrFailedPrecondition)
}

func verifyDeleteFailureValues(t *testing.T, pid int, status uint32, at time.Time) {
	if pid != 0 {
		t.Fatalf("pid expected '0' got: '%d'", pid)
	}
	if status != 0 {
		t.Fatalf("status expected '0' got: '%d'", status)
	}
	if !at.IsZero() {
		t.Fatalf("at expected 'zero' got: '%v'", at)
	}
}

func verifyDeleteSuccessValues(t *testing.T, pid int, status uint32, at time.Time, e *testShimExec) {
	if pid != e.pid {
		t.Fatalf("pid expected '%d' got: '%d'", e.pid, pid)
	}
	if status != e.status {
		t.Fatalf("status expected '%d' got: '%d'", e.status, status)
	}
	if at != e.at {
		t.Fatalf("at expected '%v' got: '%v'", e.at, at)
	}
}

func Test_lcowTask_DeleteExec_UnknownExecID_Error(t *testing.T) {
	lt, _, _ := setupTestLcowTask(t)

	pid, status, at, err := lt.DeleteExec(context.TODO(), "thisshouldnotmatch")
	verifyExpectedError(t, nil, err, errdefs.ErrNotFound)
	verifyDeleteFailureValues(t, pid, status, at)
}

func Test_lcowTask_DeleteExec_InitExecID_Unexited_Error(t *testing.T) {
	lt, _, second := setupTestLcowTask(t)
	lt.execs.Delete(second.id)

	pid, status, at, err := lt.DeleteExec(context.TODO(), "")

	verifyExpectedError(t, nil, err, errdefs.ErrFailedPrecondition)
	verifyDeleteFailureValues(t, pid, status, at)
}

func Test_lcowTask_DeleteExec_InitExecID_Unexited2ndExec_Error(t *testing.T) {
	lt, _, _ := setupTestLcowTask(t)
	pid, status, at, err := lt.DeleteExec(context.TODO(), "")

	verifyExpectedError(t, nil, err, errdefs.ErrFailedPrecondition)
	verifyDeleteFailureValues(t, pid, status, at)
}

func Test_lcowTask_DeleteExec_InitExecID_NoAdditionalExecs_Success(t *testing.T) {
	lt, init, second := setupTestLcowTask(t)
	lt.execs.Delete(second.id)
	init.Kill(context.TODO(), 0xf)

	pid, status, at, err := lt.DeleteExec(context.TODO(), "")
	if err != nil {
		t.Fatalf("should not have failed got: %v", err)
	}
	verifyDeleteSuccessValues(t, pid, status, at, init)
}

func Test_lcowTask_DeleteExec_InitExecID_Exited2ndExec_Success(t *testing.T) {
	lt, init, second := setupTestLcowTask(t)
	second.Kill(context.TODO(), 0xf)
	init.Kill(context.TODO(), 0xf)

	pid, status, at, err := lt.DeleteExec(context.TODO(), "")
	if err != nil {
		t.Fatalf("should not have failed got: %v", err)
	}
	verifyDeleteSuccessValues(t, pid, status, at, init)
}

func Test_lcowTask_DeleteExec_2ndExecID_Unexited_Error(t *testing.T) {
	lt, _, second := setupTestLcowTask(t)

	pid, status, at, err := lt.DeleteExec(context.TODO(), second.id)

	verifyExpectedError(t, nil, err, errdefs.ErrFailedPrecondition)
	verifyDeleteFailureValues(t, pid, status, at)
	_, loaded := lt.execs.Load(second.id)
	if !loaded {
		t.Fatal("delete should not have removed 2nd exec")
	}
}

func Test_lcowTask_DeleteExec_2ndExecID_Success(t *testing.T) {
	lt, _, second := setupTestLcowTask(t)
	second.Kill(context.TODO(), 0xf)

	pid, status, at, err := lt.DeleteExec(context.TODO(), second.id)

	if err != nil {
		t.Fatalf("should not have failed got: %v", err)
	}
	verifyDeleteSuccessValues(t, pid, status, at, second)
	_, loaded := lt.execs.Load(second.id)
	if loaded {
		t.Fatal("delete should have removed 2nd exec")
	}
}
