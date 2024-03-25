package auth

import (
	"context"
	"fmt"
	"strings"

	iamv1 "github.com/wongearl/go-restful-template/pkg/api/iam.ai.io/v1"
	ai "github.com/wongearl/go-restful-template/pkg/client/ai/clientset/versioned"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

type LoginRecorder interface {
	RecordLogin(username string, loginType iamv1.LoginType, provider string, sourceIP string, userAgent string, authErr error) error
}

type loginRecorder struct {
	aiClient ai.Interface
}

func NewLoginRecorder(client ai.Interface) LoginRecorder {
	return &loginRecorder{
		aiClient: client,
	}
}

func (l *loginRecorder) RecordLogin(username string, loginType iamv1.LoginType, provider string, sourceIP string, userAgent string, authErr error) error {
	// This is a temporary solution in case of user login with email,
	// '@' is not allowed in Kubernetes object name.
	username = strings.Replace(username, "@", "-", -1)

	loginEntry := &iamv1.LoginRecord{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", username),
			Labels: map[string]string{
				iamv1.UserReferenceLabel: username,
			},
		},
		Spec: iamv1.LoginRecordSpec{
			Type:      loginType,
			Provider:  provider,
			Success:   true,
			Reason:    iamv1.AuthenticatedSuccessfully,
			SourceIP:  sourceIP,
			UserAgent: userAgent,
		},
	}

	if authErr != nil {
		loginEntry.Spec.Success = false
		loginEntry.Spec.Reason = authErr.Error()
	}

	_, err := l.aiClient.IamV1().LoginRecords().Create(context.Background(), loginEntry, metav1.CreateOptions{})
	if err != nil {
		klog.Error(err)
		return err
	}
	return nil
}
