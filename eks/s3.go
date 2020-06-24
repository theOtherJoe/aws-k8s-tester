package eks

import (
	"errors"
	"os"
	"path"
	"path/filepath"

	aws_s3 "github.com/aws/aws-k8s-tester/pkg/aws/s3"
	"github.com/aws/aws-k8s-tester/pkg/fileutil"
	"go.uber.org/zap"
)

func (ts *Tester) createS3() (err error) {
	if ts.cfg.S3BucketCreate {
		if ts.cfg.S3BucketName == "" {
			return errors.New("empty S3 bucket name")
		}
		if err = aws_s3.CreateBucket(ts.lg, ts.s3API, ts.cfg.S3BucketName, ts.cfg.Region, ts.cfg.Name, ts.cfg.S3BucketLifecycleExpirationDays); err != nil {
			return err
		}
	} else {
		ts.lg.Info("skipping S3 bucket creation")
	}
	if ts.cfg.S3BucketName == "" {
		ts.lg.Info("skipping s3 bucket creation")
		return nil
	}
	return ts.cfg.Sync()
}

func (ts *Tester) deleteS3() error {
	if !ts.cfg.S3BucketCreate {
		ts.lg.Info("skipping S3 bucket deletion", zap.String("s3-bucket-name", ts.cfg.S3BucketName))
		return nil
	}
	if ts.cfg.S3BucketCreateKeep {
		ts.lg.Info("skipping S3 bucket deletion", zap.String("s3-bucket-name", ts.cfg.S3BucketName), zap.Bool("s3-bucket-create-keep", ts.cfg.S3BucketCreateKeep))
		return nil
	}
	if err := aws_s3.EmptyBucket(ts.lg, ts.s3API, ts.cfg.S3BucketName); err != nil {
		return err
	}
	return aws_s3.DeleteBucket(ts.lg, ts.s3API, ts.cfg.S3BucketName)
}

func (ts *Tester) uploadToS3() (err error) {
	if ts.cfg.S3BucketName == "" {
		ts.lg.Info("skipping s3 uploads; s3 bucket name is empty")
		return nil
	}

	if fileutil.Exist(ts.cfg.ConfigPath) {
		if err = aws_s3.Upload(
			ts.lg,
			ts.s3API,
			ts.cfg.S3BucketName,
			path.Join(ts.cfg.Name, "aws-k8s-tester-eks.config.yaml"),
			ts.cfg.ConfigPath,
		); err != nil {
			return err
		}
	}

	logFilePath := ""
	for _, fpath := range ts.cfg.LogOutputs {
		if filepath.Ext(fpath) == ".log" {
			logFilePath = fpath
			break
		}
	}
	if fileutil.Exist(logFilePath) {
		if err = aws_s3.Upload(
			ts.lg,
			ts.s3API,
			ts.cfg.S3BucketName,
			path.Join(ts.cfg.Name, "aws-k8s-tester-eks.log"),
			logFilePath,
		); err != nil {
			return err
		}
	}

	if fileutil.Exist(ts.cfg.KubeConfigPath) {
		if err = aws_s3.Upload(
			ts.lg,
			ts.s3API,
			ts.cfg.S3BucketName,
			path.Join(ts.cfg.Name, "kubeconfig.yaml"),
			ts.cfg.KubeConfigPath,
		); err != nil {
			return err
		}
	}

	if fileutil.Exist(ts.cfg.Status.ClusterMetricsRawOutputDir) {
		err = filepath.Walk(ts.cfg.Status.ClusterMetricsRawOutputDir, func(path string, info os.FileInfo, werr error) error {
			if werr != nil {
				return werr
			}
			if info.IsDir() {
				return nil
			}
			if uerr := aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				filepath.Join(ts.cfg.Name, "metrics", filepath.Base(path)),
				path,
			); uerr != nil {
				return uerr
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	if ts.cfg.IsEnabledAddOnNodeGroups() {
		if fileutil.Exist(ts.cfg.AddOnNodeGroups.RoleCFNStackYAMLFilePath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "cfn", "add-on-node-groups.role.cfn.yaml"),
				ts.cfg.AddOnNodeGroups.RoleCFNStackYAMLFilePath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnNodeGroups.NodeGroupSecurityGroupCFNStackYAMLFilePath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "cfn", "add-on-node-groups.sg.cfn.yaml"),
				ts.cfg.AddOnNodeGroups.NodeGroupSecurityGroupCFNStackYAMLFilePath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnNodeGroups.LogsTarGzPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "add-on-node-groups-logs-dir.tar.gz"),
				ts.cfg.AddOnNodeGroups.LogsTarGzPath,
			); err != nil {
				return err
			}
		}
		for asgName, cur := range ts.cfg.AddOnNodeGroups.ASGs {
			if fileutil.Exist(cur.ASGCFNStackYAMLFilePath) {
				if err = aws_s3.Upload(
					ts.lg,
					ts.s3API,
					ts.cfg.S3BucketName,
					path.Join(ts.cfg.Name, "cfn", "add-on-node-groups.asg.cfn."+asgName+".yaml"),
					cur.ASGCFNStackYAMLFilePath,
				); err != nil {
					return err
				}
			}
			if fileutil.Exist(cur.SSMDocumentCFNStackYAMLFilePath) {
				if err = aws_s3.Upload(
					ts.lg,
					ts.s3API,
					ts.cfg.S3BucketName,
					path.Join(ts.cfg.Name, "cfn", "add-on-node-groups.ssm.cfn."+asgName+".yaml"),
					cur.SSMDocumentCFNStackYAMLFilePath,
				); err != nil {
					return err
				}
			}
		}
	}

	if ts.cfg.IsEnabledAddOnManagedNodeGroups() {
		if fileutil.Exist(ts.cfg.AddOnManagedNodeGroups.RoleCFNStackYAMLFilePath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "cfn", "add-on-managed-node-groups.role.cfn.yaml"),
				ts.cfg.AddOnManagedNodeGroups.RoleCFNStackYAMLFilePath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnManagedNodeGroups.LogsTarGzPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "add-on-managed-node-groups-logs-dir.tar.gz"),
				ts.cfg.AddOnManagedNodeGroups.LogsTarGzPath,
			); err != nil {
				return err
			}
		}
		for mngName, cur := range ts.cfg.AddOnManagedNodeGroups.MNGs {
			if fileutil.Exist(cur.MNGCFNStackYAMLFilePath) {
				if err = aws_s3.Upload(
					ts.lg,
					ts.s3API,
					ts.cfg.S3BucketName,
					path.Join(ts.cfg.Name, "cfn", "add-on-managed-node-groups.mng.cfn."+mngName+".yaml"),
					cur.MNGCFNStackYAMLFilePath,
				); err != nil {
					return err
				}
			}
			if fileutil.Exist(cur.RemoteAccessSecurityCFNStackYAMLFilePath) {
				if err = aws_s3.Upload(
					ts.lg,
					ts.s3API,
					ts.cfg.S3BucketName,
					path.Join(ts.cfg.Name, "cfn", "add-on-managed-node-groups.sg.cfn."+mngName+".yaml"),
					cur.RemoteAccessSecurityCFNStackYAMLFilePath,
				); err != nil {
					return err
				}
			}
		}
	}

	if ts.cfg.IsEnabledAddOnConformance() {
		if fileutil.Exist(ts.cfg.AddOnConformance.SonobuoyResultTarGzPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "sonobuoy-result.tar.gz"),
				ts.cfg.AddOnConformance.SonobuoyResultTarGzPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnConformance.SonobuoyResultE2eLogPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "sonobuoy-result.e2e.log"),
				ts.cfg.AddOnConformance.SonobuoyResultE2eLogPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnConformance.SonobuoyResultJunitXMLPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "sonobuoy-result.junit.xml"),
				ts.cfg.AddOnConformance.SonobuoyResultJunitXMLPath,
			); err != nil {
				return err
			}
		}
	}

	if ts.cfg.IsEnabledAddOnAppMesh() {
		if fileutil.Exist(ts.cfg.AddOnAppMesh.PolicyCFNStackYAMLFilePath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "cfn", "add-on-app-mesh.policy.cfn.yaml"),
				ts.cfg.AddOnAppMesh.PolicyCFNStackYAMLFilePath,
			); err != nil {
				return err
			}
		}
	}

	if ts.cfg.IsEnabledAddOnCSRsLocal() {
		if fileutil.Exist(ts.cfg.AddOnCSRsLocal.RequestsWritesRawJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "csrs-local-requests-writes-raw.json"),
				ts.cfg.AddOnCSRsLocal.RequestsWritesRawJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnCSRsLocal.RequestsWritesSummaryJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "csrs-local-requests-writes-summary.json"),
				ts.cfg.AddOnCSRsLocal.RequestsWritesSummaryJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnCSRsLocal.RequestsWritesSummaryTablePath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "csrs-local-requests-writes-summary.txt"),
				ts.cfg.AddOnCSRsLocal.RequestsWritesSummaryTablePath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnCSRsLocal.RequestsWritesCompareJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "csrs-local-requests-writes-compare.json"),
				ts.cfg.AddOnCSRsLocal.RequestsWritesCompareJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnCSRsLocal.RequestsWritesCompareTablePath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "csrs-local-requests-writes-compare.txt"),
				ts.cfg.AddOnCSRsLocal.RequestsWritesCompareTablePath,
			); err != nil {
				return err
			}
		}
	}

	if ts.cfg.IsEnabledAddOnCSRsRemote() {
		if fileutil.Exist(ts.cfg.AddOnCSRsRemote.RequestsWritesRawJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "csrs-remote-requests-writes-raw.json"),
				ts.cfg.AddOnCSRsRemote.RequestsWritesRawJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnCSRsRemote.RequestsWritesSummaryJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "csrs-remote-requests-writes-summary.json"),
				ts.cfg.AddOnCSRsRemote.RequestsWritesSummaryJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnCSRsRemote.RequestsWritesSummaryTablePath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "csrs-remote-requests-writes-summary.txt"),
				ts.cfg.AddOnCSRsRemote.RequestsWritesSummaryTablePath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnCSRsRemote.RequestsWritesCompareJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "csrs-remote-requests-writes-compare.json"),
				ts.cfg.AddOnCSRsRemote.RequestsWritesCompareJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnCSRsRemote.RequestsWritesCompareTablePath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "csrs-remote-requests-writes-compare.txt"),
				ts.cfg.AddOnCSRsRemote.RequestsWritesCompareTablePath,
			); err != nil {
				return err
			}
		}
	}

	if ts.cfg.IsEnabledAddOnConfigmapsLocal() {
		if fileutil.Exist(ts.cfg.AddOnConfigmapsLocal.RequestsWritesRawJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "configmaps-local-requests-writes-raw.json"),
				ts.cfg.AddOnConfigmapsLocal.RequestsWritesRawJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnConfigmapsLocal.RequestsWritesSummaryJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "configmaps-local-requests-writes-summary.json"),
				ts.cfg.AddOnConfigmapsLocal.RequestsWritesSummaryJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnConfigmapsLocal.RequestsWritesSummaryTablePath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "configmaps-local-requests-writes-summary.txt"),
				ts.cfg.AddOnConfigmapsLocal.RequestsWritesSummaryTablePath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnConfigmapsLocal.RequestsWritesCompareJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "configmaps-local-requests-writes-compare.json"),
				ts.cfg.AddOnConfigmapsLocal.RequestsWritesCompareJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnConfigmapsLocal.RequestsWritesCompareTablePath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "configmaps-local-requests-writes-compare.txt"),
				ts.cfg.AddOnConfigmapsLocal.RequestsWritesCompareTablePath,
			); err != nil {
				return err
			}
		}
	}

	if ts.cfg.IsEnabledAddOnConfigmapsRemote() {
		if fileutil.Exist(ts.cfg.AddOnConfigmapsRemote.RequestsWritesRawJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "configmaps-remote-requests-writes-raw.json"),
				ts.cfg.AddOnConfigmapsRemote.RequestsWritesRawJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnConfigmapsRemote.RequestsWritesSummaryJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "configmaps-remote-requests-writes-summary.json"),
				ts.cfg.AddOnConfigmapsRemote.RequestsWritesSummaryJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnConfigmapsRemote.RequestsWritesSummaryTablePath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "configmaps-remote-requests-writes-summary.txt"),
				ts.cfg.AddOnConfigmapsRemote.RequestsWritesSummaryTablePath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnConfigmapsRemote.RequestsWritesCompareJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "configmaps-remote-requests-writes-compare.json"),
				ts.cfg.AddOnConfigmapsRemote.RequestsWritesCompareJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnConfigmapsRemote.RequestsWritesCompareTablePath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "configmaps-remote-requests-writes-compare.txt"),
				ts.cfg.AddOnConfigmapsRemote.RequestsWritesCompareTablePath,
			); err != nil {
				return err
			}
		}
	}

	if ts.cfg.IsEnabledAddOnSecretsLocal() {
		if fileutil.Exist(ts.cfg.AddOnSecretsLocal.RequestsWritesRawJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "secrets-local-requests-writes-raw.json"),
				ts.cfg.AddOnSecretsLocal.RequestsWritesRawJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnSecretsLocal.RequestsWritesSummaryJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "secrets-local-requests-writes-summary.json"),
				ts.cfg.AddOnSecretsLocal.RequestsWritesSummaryJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnSecretsLocal.RequestsWritesSummaryTablePath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "secrets-local-requests-writes-summary.txt"),
				ts.cfg.AddOnSecretsLocal.RequestsWritesSummaryTablePath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnSecretsLocal.RequestsWritesCompareJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "secrets-local-requests-writes-compare.json"),
				ts.cfg.AddOnSecretsLocal.RequestsWritesCompareJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnSecretsLocal.RequestsWritesCompareTablePath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "secrets-local-requests-writes-compare.txt"),
				ts.cfg.AddOnSecretsLocal.RequestsWritesCompareTablePath,
			); err != nil {
				return err
			}
		}

		if fileutil.Exist(ts.cfg.AddOnSecretsLocal.RequestsReadsRawJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "secrets-local-requests-reads-raw.json"),
				ts.cfg.AddOnSecretsLocal.RequestsReadsRawJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnSecretsLocal.RequestsReadsSummaryJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "secrets-local-requests-reads-summary.json"),
				ts.cfg.AddOnSecretsLocal.RequestsReadsSummaryJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnSecretsLocal.RequestsReadsSummaryTablePath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "secrets-local-requests-reads-summary.txt"),
				ts.cfg.AddOnSecretsLocal.RequestsReadsSummaryTablePath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnSecretsLocal.RequestsReadsCompareJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "secrets-local-requests-reads-compare.json"),
				ts.cfg.AddOnSecretsLocal.RequestsReadsCompareJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnSecretsLocal.RequestsReadsCompareTablePath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "secrets-local-requests-reads-compare.txt"),
				ts.cfg.AddOnSecretsLocal.RequestsReadsCompareTablePath,
			); err != nil {
				return err
			}
		}
	}

	if ts.cfg.IsEnabledAddOnSecretsRemote() {
		if fileutil.Exist(ts.cfg.AddOnSecretsRemote.RequestsWritesRawJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "secrets-remote-requests-writes-raw.json"),
				ts.cfg.AddOnSecretsRemote.RequestsWritesRawJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnSecretsRemote.RequestsWritesSummaryJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "secrets-remote-requests-writes-summary.json"),
				ts.cfg.AddOnSecretsRemote.RequestsWritesSummaryJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnSecretsRemote.RequestsWritesSummaryTablePath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "secrets-remote-requests-writes-summary.txt"),
				ts.cfg.AddOnSecretsRemote.RequestsWritesSummaryTablePath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnSecretsRemote.RequestsWritesCompareJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "secrets-remote-requests-writes-compare.json"),
				ts.cfg.AddOnSecretsRemote.RequestsWritesCompareJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnSecretsRemote.RequestsWritesCompareTablePath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "secrets-remote-requests-writes-compare.txt"),
				ts.cfg.AddOnSecretsRemote.RequestsWritesCompareTablePath,
			); err != nil {
				return err
			}
		}

		if fileutil.Exist(ts.cfg.AddOnSecretsRemote.RequestsReadsRawJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "secrets-remote-requests-reads-raw.json"),
				ts.cfg.AddOnSecretsRemote.RequestsReadsRawJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnSecretsRemote.RequestsReadsSummaryJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "secrets-remote-requests-reads-summary.json"),
				ts.cfg.AddOnSecretsRemote.RequestsReadsSummaryJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnSecretsRemote.RequestsReadsSummaryTablePath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "secrets-remote-requests-reads-summary.txt"),
				ts.cfg.AddOnSecretsRemote.RequestsReadsSummaryTablePath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnSecretsRemote.RequestsReadsCompareJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "secrets-remote-requests-reads-compare.json"),
				ts.cfg.AddOnSecretsRemote.RequestsReadsCompareJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnSecretsRemote.RequestsReadsCompareTablePath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "secrets-remote-requests-reads-compare.txt"),
				ts.cfg.AddOnSecretsRemote.RequestsReadsCompareTablePath,
			); err != nil {
				return err
			}
		}
	}

	if ts.cfg.IsEnabledAddOnFargate() {
		if fileutil.Exist(ts.cfg.AddOnFargate.RoleCFNStackYAMLFilePath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "cfn", "add-on-fargate.role.cfn.yaml"),
				ts.cfg.AddOnFargate.RoleCFNStackYAMLFilePath,
			); err != nil {
				return err
			}
		}
	}

	if ts.cfg.IsEnabledAddOnIRSA() {
		if fileutil.Exist(ts.cfg.AddOnIRSA.RoleCFNStackYAMLFilePath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "cfn", "add-on-irsa.role.cfn.yaml"),
				ts.cfg.AddOnIRSA.RoleCFNStackYAMLFilePath,
			); err != nil {
				return err
			}
		}
	}

	if ts.cfg.IsEnabledAddOnIRSAFargate() {
		if fileutil.Exist(ts.cfg.AddOnIRSAFargate.RoleCFNStackYAMLFilePath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "cfn", "add-on-irsa-fargate.role.cfn.yaml"),
				ts.cfg.AddOnIRSAFargate.RoleCFNStackYAMLFilePath,
			); err != nil {
				return err
			}
		}
	}

	if ts.cfg.IsEnabledAddOnClusterLoaderLocal() {
		if fileutil.Exist(ts.cfg.AddOnClusterLoaderLocal.ReportTarGzPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "cluster-loader-local.tar.gz"),
				ts.cfg.AddOnClusterLoaderLocal.ReportTarGzPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnClusterLoaderLocal.LogPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "cluster-loader-local.log"),
				ts.cfg.AddOnClusterLoaderLocal.LogPath,
			); err != nil {
				return err
			}
		}
	}

	if ts.cfg.IsEnabledAddOnClusterLoaderRemote() {
		if fileutil.Exist(ts.cfg.AddOnClusterLoaderRemote.ReportTarGzPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "cluster-loader-remote.tar.gz"),
				ts.cfg.AddOnClusterLoaderRemote.ReportTarGzPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnClusterLoaderRemote.LogPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "cluster-loader-remote.log"),
				ts.cfg.AddOnClusterLoaderRemote.LogPath,
			); err != nil {
				return err
			}
		}
	}

	if ts.cfg.IsEnabledAddOnStresserLocal() {
		if fileutil.Exist(ts.cfg.AddOnStresserLocal.RequestsWritesRawJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "stresser-local-requests-writes-raw.json"),
				ts.cfg.AddOnStresserLocal.RequestsWritesRawJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnStresserLocal.RequestsWritesSummaryJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "stresser-local-requests-writes-summary.json"),
				ts.cfg.AddOnStresserLocal.RequestsWritesSummaryJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnStresserLocal.RequestsWritesSummaryTablePath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "stresser-local-requests-writes-summary.txt"),
				ts.cfg.AddOnStresserLocal.RequestsWritesSummaryTablePath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnStresserLocal.RequestsWritesCompareJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "stresser-local-requests-writes-compare.json"),
				ts.cfg.AddOnStresserLocal.RequestsWritesCompareJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnStresserLocal.RequestsWritesCompareTablePath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "stresser-local-requests-writes-compare.txt"),
				ts.cfg.AddOnStresserLocal.RequestsWritesCompareTablePath,
			); err != nil {
				return err
			}
		}

		if fileutil.Exist(ts.cfg.AddOnStresserLocal.RequestsReadsRawJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "stresser-local-requests-reads-raw.json"),
				ts.cfg.AddOnStresserLocal.RequestsReadsRawJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnStresserLocal.RequestsReadsSummaryJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "stresser-local-requests-reads-summary.json"),
				ts.cfg.AddOnStresserLocal.RequestsReadsSummaryJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnStresserLocal.RequestsReadsSummaryTablePath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "stresser-local-requests-reads-summary.txt"),
				ts.cfg.AddOnStresserLocal.RequestsReadsSummaryTablePath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnStresserLocal.RequestsReadsCompareJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "stresser-local-requests-reads-compare.json"),
				ts.cfg.AddOnStresserLocal.RequestsReadsCompareJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnStresserLocal.RequestsReadsCompareTablePath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "stresser-local-requests-reads-compare.txt"),
				ts.cfg.AddOnStresserLocal.RequestsReadsCompareTablePath,
			); err != nil {
				return err
			}
		}
	}

	if ts.cfg.IsEnabledAddOnStresserRemote() {
		if fileutil.Exist(ts.cfg.AddOnStresserRemote.RequestsWritesRawJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "stresser-remote-requests-writes-raw.json"),
				ts.cfg.AddOnStresserRemote.RequestsWritesRawJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnStresserRemote.RequestsWritesSummaryJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "stresser-remote-requests-writes-summary.json"),
				ts.cfg.AddOnStresserRemote.RequestsWritesSummaryJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnStresserRemote.RequestsWritesSummaryTablePath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "stresser-remote-requests-writes-summary.txt"),
				ts.cfg.AddOnStresserRemote.RequestsWritesSummaryTablePath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnStresserRemote.RequestsWritesCompareJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "stresser-remote-requests-writes-compare.json"),
				ts.cfg.AddOnStresserRemote.RequestsWritesCompareJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnStresserRemote.RequestsWritesCompareTablePath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "stresser-remote-requests-writes-compare.txt"),
				ts.cfg.AddOnStresserRemote.RequestsWritesCompareTablePath,
			); err != nil {
				return err
			}
		}

		if fileutil.Exist(ts.cfg.AddOnStresserRemote.RequestsReadsRawJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "stresser-remote-requests-reads-raw.json"),
				ts.cfg.AddOnStresserRemote.RequestsReadsRawJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnStresserRemote.RequestsReadsSummaryJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "stresser-remote-requests-reads-summary.json"),
				ts.cfg.AddOnStresserRemote.RequestsReadsSummaryJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnStresserRemote.RequestsReadsSummaryTablePath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "stresser-remote-requests-reads-summary.txt"),
				ts.cfg.AddOnStresserRemote.RequestsReadsSummaryTablePath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnStresserRemote.RequestsReadsCompareJSONPath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "stresser-remote-requests-reads-compare.json"),
				ts.cfg.AddOnStresserRemote.RequestsReadsCompareJSONPath,
			); err != nil {
				return err
			}
		}
		if fileutil.Exist(ts.cfg.AddOnStresserRemote.RequestsReadsCompareTablePath) {
			if err = aws_s3.Upload(
				ts.lg,
				ts.s3API,
				ts.cfg.S3BucketName,
				path.Join(ts.cfg.Name, "stresser-remote-requests-reads-compare.txt"),
				ts.cfg.AddOnStresserRemote.RequestsReadsCompareTablePath,
			); err != nil {
				return err
			}
		}
	}

	return err
}
