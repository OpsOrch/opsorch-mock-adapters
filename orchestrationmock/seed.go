package orchestrationmock

import (
	"time"

	"github.com/opsorch/opsorch-core/schema"
)

func (p *Provider) seed() {
	now := time.Now().UTC()

	// Seed playbook plans
	p.seedPlaybooks(now)

	// Seed runbook plans
	p.seedRunbooks(now)

	// Seed release checklist plans
	p.seedReleaseChecklists(now)

	// Seed complex flow plans
	p.seedComplexFlows(now)

	// Seed active runs
	p.seedRuns(now)
}

func (p *Provider) seedPlaybooks(now time.Time) {
	playbooks := []schema.OrchestrationPlan{
		{
			ID:    "plan-playbook-001",
			Title: "Database Connection Pool Exhaustion",
			Description: "Diagnostic and mitigation steps for database connection pool exhaustion incidents. " +
				"Use this response for 'Cascading Failure - Database Connection Pool Exhaustion' scenarios or saturation alerts.",
			Steps: []schema.OrchestrationStep{
				{
					ID:    "step-1",
					Title: "Diagnose connection pool status",
					Type:  "manual",
					Description: "Check current connection pool metrics and identify saturation levels. " +
						"Run: `SELECT count(*) FROM pg_stat_activity;`",
				},
				{
					ID:    "step-2",
					Title: "Identify connection leaks",
					Type:  "manual",
					Description: "Analyze long-running queries and idle connections. " +
						"Look for queries stuck in transaction state.",
					DependsOn: []string{"step-1"},
				},
				{
					ID:    "step-3",
					Title: "Notify affected services",
					Type:  "manual",
					Description: "Send notification to service owners about potential connection issues. " +
						"Prepare for possible service degradation.",
					DependsOn: []string{"step-2"},
				},
				{
					ID:    "step-4",
					Title: "Restart connection pool",
					Type:  "manual",
					Description: "Gracefully restart the connection pool to clear leaked connections. " +
						"Coordinate with on-call to minimize impact.",
					DependsOn: []string{"step-3"},
				},
				{
					ID:    "step-5",
					Title: "Scale connection limits",
					Type:  "manual",
					Description: "Increase connection pool size if needed to handle current load. " +
						"Update configuration and redeploy.",
					DependsOn: []string{"step-4"},
				},
				{
					ID:    "step-6",
					Title: "Verify recovery",
					Type:  "manual",
					Description: "Confirm connection pool is healthy and services are responding normally. " +
						"Monitor for 30 minutes.",
					DependsOn: []string{"step-5"},
				},
			},
			URL:     "https://runbook.demo/playbooks/db-connection-pool",
			Version: "1.0",
			Tags: map[string]string{
				"type":        "playbook",
				"service":     "svc-database",
				"environment": "prod",
			},
			Metadata: map[string]any{
				"source":      p.cfg.Source,
				"runbook_url": "https://runbook.demo/db-connection-pool",
				"severity":    "sev1",
				"team":        "platform",
			},
		},
		{
			ID:    "plan-playbook-002",
			Title: "High Latency Investigation",
			Description: "Systematic investigation and mitigation of high latency issues, such as 'Checkout latency impacting EU customers'. " +
				"Includes observability checks, bottleneck investigation, and escalation procedures.",
			Steps: []schema.OrchestrationStep{
				{
					ID:    "step-1",
					Title: "Check application metrics",
					Type:  "manual",
					Description: "Review p50, p95, p99 latency metrics across all services. " +
						"Identify which services are affected.",
				},
				{
					ID:    "step-2",
					Title: "Trace request flow",
					Type:  "manual",
					Description: "Use distributed tracing to identify where latency is introduced. " +
						"Check database, cache, and external service calls.",
					DependsOn: []string{"step-1"},
				},
				{
					ID:          "step-2a",
					Title:       "Check DB Latency",
					Type:        "manual",
					Description: "Analyze database query performance and wait events.",
					DependsOn:   []string{"step-1"},
				},
				{
					ID:          "step-2b",
					Title:       "Check Cache Hit Rate",
					Type:        "manual",
					Description: "Verify Redis/Memcached hit rates and eviction metrics.",
					DependsOn:   []string{"step-1"},
				},
				{
					ID:          "step-2c",
					Title:       "Check Upstream Services",
					Type:        "manual",
					Description: "Review latency metrics for dependent microservices.",
					DependsOn:   []string{"step-1"},
				},
				{
					ID:    "step-3",
					Title: "Identify bottleneck",
					Type:  "manual",
					Description: "Determine root cause based on parallel investigation results. " +
						"Gather evidence for mitigation.",
					DependsOn: []string{"step-2", "step-2a", "step-2b", "step-2c"},
				},
				{
					ID:    "step-4",
					Title: "Apply mitigation",
					Type:  "manual",
					Description: "Execute targeted fix based on identified bottleneck. " +
						"May include query optimization, cache warming, or scaling.",
					DependsOn: []string{"step-3"},
				},
				{
					ID:    "step-5",
					Title: "Escalate if needed",
					Type:  "manual",
					Description: "If mitigation unsuccessful, escalate to platform team or external vendor. " +
						"Prepare incident summary.",
					DependsOn: []string{"step-4"},
				},
			},
			URL:     "https://runbook.demo/playbooks/high-latency",
			Version: "1.0",
			Tags: map[string]string{
				"type":        "playbook",
				"environment": "prod",
			},
			Metadata: map[string]any{
				"source":      p.cfg.Source,
				"runbook_url": "https://runbook.demo/high-latency",
				"severity":    "sev2",
			},
		},
		{
			ID:    "plan-playbook-003",
			Title: "Service Degradation Response",
			Description: "Coordinated response to service degradation incidents. " +
				"Includes triage, impact assessment, communication, and mitigation.",
			Steps: []schema.OrchestrationStep{
				{
					ID:    "step-1",
					Title: "Triage incident",
					Type:  "manual",
					Description: "Assess severity and scope of degradation. " +
						"Determine if incident or maintenance window.",
				},
				{
					ID:    "step-2",
					Title: "Assess customer impact",
					Type:  "manual",
					Description: "Quantify affected users and business impact. " +
						"Check error rates and transaction volume.",
					DependsOn: []string{"step-1"},
				},
				{
					ID:    "step-3",
					Title: "Communicate status",
					Type:  "manual",
					Description: "Post status page update and notify stakeholders. " +
						"Provide ETA for resolution.",
					DependsOn: []string{"step-2"},
				},
				{
					ID:    "step-4",
					Title: "Execute mitigation",
					Type:  "manual",
					Description: "Apply fix or workaround to restore service. " +
						"May include rollback, scaling, or failover.",
					DependsOn: []string{"step-3"},
				},
				{
					ID:    "step-5",
					Title: "Schedule postmortem",
					Type:  "manual",
					Description: "Schedule postmortem meeting to review root cause and prevention measures. " +
						"Assign action items.",
					DependsOn: []string{"step-4"},
				},
			},
			URL:     "https://runbook.demo/playbooks/service-degradation",
			Version: "1.0",
			Tags: map[string]string{
				"type":        "playbook",
				"environment": "prod",
			},
			Metadata: map[string]any{
				"source":      p.cfg.Source,
				"runbook_url": "https://runbook.demo/service-degradation",
				"severity":    "sev2",
			},
		},
		{
			ID:    "plan-playbook-004",
			Title: "Suspicious Activity Protocol",
			Description: "Security response for account anomalies. " +
				"Triggered by 'Unusual authentication pattern detected' alerts or credential stuffing signals.",
			Steps: []schema.OrchestrationStep{
				{
					ID:    "step-1",
					Title: "Analyze GeoIP patterns",
					Type:  "manual",
					Description: "Review login IP distribution and calculate distance velocity. " +
						"Identify impossible travel.",
				},
				{
					ID:    "step-2",
					Title: "Lock compromised accounts",
					Type:  "manual",
					Description: "Temporarily lock accounts with confirmed suspicious activity. " +
						"Revoke active sessions.",
					DependsOn: []string{"step-1"},
				},
				{
					ID:    "step-2b",
					Title: "Force MFA challenge",
					Type:  "manual",
					Description: "Require MFA for next login on borderline accounts. " +
						"Reset 2FA tokens if compromised.",
					DependsOn: []string{"step-1"},
				},
				{
					ID:    "step-3",
					Title: "Notify users",
					Type:  "manual",
					Description: "Send security alerts to affected users via email/SMS. " +
						"Prompt for password change.",
					DependsOn: []string{"step-2", "step-2b"},
				},
			},
			URL:     "https://runbook.demo/playbooks/security-incident",
			Version: "1.0",
			Tags: map[string]string{
				"type":        "playbook",
				"team":        "security",
				"environment": "prod",
			},
			Metadata: map[string]any{
				"source":      p.cfg.Source,
				"runbook_url": "https://runbook.demo/security-incident",
				"severity":    "sev1",
			},
		},
		{
			ID:    "plan-playbook-005",
			Title: "Analytics Correlation Runbook",
			Description: "Automated response to analytics correlation lag. " +
				"Includes backfilling data and verifying integrity.",
			Steps: []schema.OrchestrationStep{
				{
					ID:    "step-1",
					Title: "Observe Correlation Lag",
					Type:  "automated",
					Description: "Check the current lag metrics from the analytics pipeline. " +
						"Automated check.",
				},
				{
					ID:    "step-2",
					Title: "Backfill Missing Data",
					Type:  "automated",
					Description: "Trigger the backfill job for the affected time range. " +
						"Automated action.",
					DependsOn: []string{"step-1"},
				},
				{
					ID:    "step-3",
					Title: "Verify Data Integrity",
					Type:  "automated",
					Description: "Run checksum validation on the backfilled data. " +
						"Automated verification.",
					DependsOn: []string{"step-2"},
				},
				{
					ID:    "step-4",
					Title: "Notify Team",
					Type:  "manual",
					Description: "Notify the data platform team about the incident and resolution. " +
						"Manual step to ensure visibility.",
					DependsOn: []string{"step-3"},
				},
			},
			URL:     "https://runbook.demo/playbooks/analytics-correlation",
			Version: "1.0",
			Tags: map[string]string{
				"type":        "playbook",
				"team":        "data-platform",
				"service":     "svc-analytics",
				"environment": "prod",
			},
			Metadata: map[string]any{
				"source":      p.cfg.Source,
				"runbook_url": "https://runbook.demo/analytics-correlation",
				"severity":    "sev2",
			},
		},
	}

	for _, plan := range playbooks {
		p.plans[plan.ID] = plan
	}
}

func (p *Provider) seedRunbooks(now time.Time) {
	runbooks := []schema.OrchestrationPlan{
		{
			ID:    "plan-runbook-001",
			Title: "Database Failover",
			Description: "Procedure for failing over to standby database. " +
				"Trigger this runbook when 'Database primary failover initiated' alert fires.",
			Steps: []schema.OrchestrationStep{
				{
					ID:    "step-1",
					Title: "Pre-failover checks",
					Type:  "manual",
					Description: "Verify standby database is healthy and in sync. " +
						"Check replication lag and disk space.",
				},
				{
					ID:    "step-2",
					Title: "Notify stakeholders",
					Type:  "manual",
					Description: "Send notification to all service owners about planned failover. " +
						"Provide maintenance window details.",
					DependsOn: []string{"step-1"},
				},
				{
					ID:          "step-3a",
					Title:       "Stop Background Jobs",
					Type:        "manual",
					Description: "Pause all cron jobs and worker queues to prevent data inconsistency.",
					DependsOn:   []string{"step-2"},
				},
				{
					ID:          "step-3b",
					Title:       "Prepare Standby",
					Type:        "manual",
					Description: "Ensure standby is caught up and ready for promotion.",
					DependsOn:   []string{"step-2"},
				},
				{
					ID:    "step-4",
					Title: "Execute failover",
					Type:  "manual",
					Description: "Promote standby to primary and update connection strings. " +
						"Run: `pg_ctl promote -D /var/lib/postgresql/data`",
					DependsOn: []string{"step-3a", "step-3b"},
				},
				{
					ID:    "step-5",
					Title: "Validate new primary",
					Type:  "manual",
					Description: "Confirm new primary is accepting connections and serving queries. " +
						"Run smoke tests.",
					DependsOn: []string{"step-4"},
				},
				{
					ID:    "step-6",
					Title: "Update DNS records",
					Type:  "manual",
					Description: "Update DNS to point to new primary database. " +
						"Wait for TTL to expire.",
					DependsOn: []string{"step-5"},
				},
			},
			URL:     "https://runbook.demo/runbooks/db-failover",
			Version: "1.0",
			Tags: map[string]string{
				"type":        "runbook",
				"service":     "svc-database",
				"environment": "prod",
			},
			Metadata: map[string]any{
				"source":      p.cfg.Source,
				"runbook_url": "https://runbook.demo/db-failover",
				"team":        "platform",
			},
		},
		{
			ID:    "plan-runbook-002",
			Title: "Certificate Rotation",
			Description: "Procedure for rotating SSL/TLS certificates. " +
				"Recommended response for 'Certificate expiration warning' alerts.",
			Steps: []schema.OrchestrationStep{
				{
					ID:    "step-1",
					Title: "Backup current certificates",
					Type:  "manual",
					Description: "Create backup of current certificates and keys. " +
						"Store in secure location.",
				},
				{
					ID:    "step-2",
					Title: "Generate new certificates",
					Type:  "manual",
					Description: "Generate new certificates from CA. " +
						"Run: `certbot renew --force-renewal`",
					DependsOn: []string{"step-1"},
				},
				{
					ID:          "step-3a",
					Title:       "Deploy to LBs",
					Type:        "manual",
					Description: "Update SSL certificates on HAProxy/ALB endpoints.",
					DependsOn:   []string{"step-2"},
				},
				{
					ID:          "step-3b",
					Title:       "Deploy to App Servers",
					Type:        "manual",
					Description: "Distribute certificates to backend application instances.",
					DependsOn:   []string{"step-2"},
				},
				{
					ID:    "step-4",
					Title: "Verify certificate validity",
					Type:  "manual",
					Description: "Verify new certificates are valid and properly installed. " +
						"Check expiration dates and certificate chain.",
					DependsOn: []string{"step-3a", "step-3b"},
				},
			},
			URL:     "https://runbook.demo/runbooks/cert-rotation",
			Version: "1.0",
			Tags: map[string]string{
				"type":        "runbook",
				"environment": "prod",
			},
			Metadata: map[string]any{
				"source":      p.cfg.Source,
				"runbook_url": "https://runbook.demo/cert-rotation",
				"team":        "platform",
			},
		},
		{
			ID:    "plan-runbook-003",
			Title: "Cache Flush and Warmup",
			Description: "Procedure for flushing and warming up cache. " +
				"Use to resolve 'Cache hit rate degradation' alerts or 'Redis cache hit rate dropped' warnings.",
			Steps: []schema.OrchestrationStep{
				{
					ID:    "step-1",
					Title: "Drain cache connections",
					Type:  "manual",
					Description: "Gracefully drain existing cache connections. " +
						"Stop accepting new connections.",
				},
				{
					ID:    "step-2",
					Title: "Flush cache data",
					Type:  "manual",
					Description: "Clear all data from cache. " +
						"Run: `redis-cli FLUSHALL`",
					DependsOn: []string{"step-1"},
				},
				{
					ID:          "step-3a",
					Title:       "Warmup User Data",
					Type:        "manual",
					Description: "Pre-load active user profiles into cache.",
					DependsOn:   []string{"step-2"},
				},
				{
					ID:          "step-3b",
					Title:       "Warmup Product Catalog",
					Type:        "manual",
					Description: "Pre-load high-traffic product listings.",
					DependsOn:   []string{"step-2"},
				},
				{
					ID:    "step-4",
					Title: "Verify cache health",
					Type:  "manual",
					Description: "Confirm cache is responding and hit rates are normal. " +
						"Monitor for 15 minutes.",
					DependsOn: []string{"step-3a", "step-3b"},
				},
			},
			URL:     "https://runbook.demo/runbooks/cache-flush",
			Version: "1.0",
			Tags: map[string]string{
				"type":        "runbook",
				"service":     "svc-cache",
				"environment": "prod",
			},
			Metadata: map[string]any{
				"source":      p.cfg.Source,
				"runbook_url": "https://runbook.demo/cache-flush",
				"team":        "platform",
			},
		},
		{
			ID:    "plan-runbook-004",
			Title: "Pod Restart Analysis",
			Description: "Diagnostic workflow for unstable containers. " +
				"Response for 'Container restart loop on pod' alerts or OOMKilled events.",
			Steps: []schema.OrchestrationStep{
				{
					ID:    "step-1",
					Title: "Fetch pod logs",
					Type:  "manual",
					Description: "Retrieve previous container logs to identify crash reason. " +
						"Look for panic stack traces.",
				},
				{
					ID:    "step-2",
					Title: "Check resource limits",
					Type:  "manual",
					Description: "Compare memory usage vs configured limits. " +
						"Identify memory leaks or under-provisioning.",
					DependsOn: []string{"step-1"},
				},
				{
					ID:    "step-3",
					Title: "Redeploy pod",
					Type:  "manual",
					Description: "Delete pod to force reschedule on fresh node. " +
						"Validate startup health.",
					DependsOn: []string{"step-2"},
				},
			},
			URL:     "https://runbook.demo/runbooks/pod-restart",
			Version: "1.0",
			Tags: map[string]string{
				"type":        "runbook",
				"service":     "svc-kubernetes",
				"environment": "prod",
			},
			Metadata: map[string]any{
				"source":      p.cfg.Source,
				"runbook_url": "https://runbook.demo/pod-restart",
				"team":        "platform",
			},
		},
		{
			ID:    "plan-runbook-005",
			Title: "API Rate Limit Mitigation",
			Description: "Mitigation for 'API rate limit exhaustion' alerts. " +
				"Managing quotas for high-volume consumers.",
			Steps: []schema.OrchestrationStep{
				{
					ID:    "step-1",
					Title: "Identify top consumers",
					Type:  "manual",
					Description: "Analyze request logs to find IP/API keys exceeding quota. " +
						"Run: `SELECT client_id, count(*) FROM api_logs GROUP BY client_id`",
				},
				{
					ID:          "step-2a",
					Title:       "Increase temporary quota",
					Type:        "manual",
					Description: "Approve temporary limit increase for legitimate traffic spike.",
					DependsOn:   []string{"step-1"},
				},
				{
					ID:          "step-2b",
					Title:       "Ban abusive client",
					Type:        "manual",
					Description: "Block API key or IP address if traffic is malicious/bot.",
					DependsOn:   []string{"step-1"},
				},
				{
					ID:    "step-3",
					Title: "Verify traffic normalized",
					Type:  "manual",
					Description: "Confirm error rates dropped and latency recovered. " +
						"Monitor for 10 minutes.",
					DependsOn: []string{"step-2a", "step-2b"},
				},
			},
			URL:     "https://runbook.demo/runbooks/rate-limits",
			Version: "1.0",
			Tags: map[string]string{
				"type":        "runbook",
				"service":     "svc-gateway",
				"environment": "prod",
			},
			Metadata: map[string]any{
				"source":      p.cfg.Source,
				"runbook_url": "https://runbook.demo/rate-limits",
				"team":        "platform",
			},
		},
		{
			ID:    "plan-runbook-006",
			Title: "Catalog Sync Repair",
			Description: "Resolution for 'Catalog inventory sync drift' alerts. " +
				"Fixes inconsistencies between ERP and Store catalog.",
			Steps: []schema.OrchestrationStep{
				{
					ID:          "step-1",
					Title:       "Check drift metrics",
					Type:        "manual",
					Description: "Compare item counts and timestamps between ERP and Catalog DB.",
				},
				{
					ID:    "step-2",
					Title: "Trigger full sync",
					Type:  "manual",
					Description: "Force a full reconciliation job to overwrite stale data. " +
						"Action: `POST /admin/sync/full`",
					DependsOn: []string{"step-1"},
				},
				{
					ID:          "step-3",
					Title:       "Verify record counts",
					Type:        "manual",
					Description: "Confirm drift metric is zero after job completion.",
					DependsOn:   []string{"step-2"},
				},
			},
			URL:     "https://runbook.demo/runbooks/catalog-sync",
			Version: "1.0",
			Tags: map[string]string{
				"type":        "runbook",
				"service":     "svc-catalog",
				"environment": "prod",
			},
			Metadata: map[string]any{
				"source":      p.cfg.Source,
				"runbook_url": "https://runbook.demo/catalog-sync",
				"team":        "data",
			},
		},
		{
			ID:    "plan-runbook-007",
			Title: "Payment Latency Mitigation",
			Description: "Diagnostic and remediation steps for payment service performance issues. " +
				"Used for high P99 latency alerts.",
			Steps: []schema.OrchestrationStep{
				{
					ID:    "step-1",
					Title: "Check payment gateway metrics",
					Type:  "automated",
					Description: "Review latency and error rates for third-party payment gateways. " +
						"Identify if issue is external.",
				},
				{
					ID:    "step-2",
					Title: "Analyze upstream dependencies",
					Type:  "manual",
					Description: "Check latency metrics for internal services like Fraud and Inventory. " +
						"Identify downstream bottlenecks.",
					DependsOn: []string{"step-1"},
				},
				{
					ID:    "step-3",
					Title: "Increase gateway timeout",
					Type:  "manual",
					Description: "Update payment gateway timeout configuration to handle transient spikes. " +
						"Deploy config change.",
					DependsOn: []string{"step-2"},
				},
				{
					ID:    "step-4",
					Title: "Scale payment service",
					Type:  "manual",
					Description: "Increase horizontal pod autoscaling targets for payment-service. " +
						"Ensure enough capacity.",
					DependsOn: []string{"step-3"},
				},
				{
					ID:    "step-5",
					Title: "Verify latency recovery",
					Type:  "automated",
					Description: "Monitor P99 latency metrics for 15 minutes. " +
						"Confirm stability.",
					DependsOn: []string{"step-4"},
				},
			},
			URL:     "https://runbook.demo/runbooks/payment-latency",
			Version: "1.0",
			Tags: map[string]string{
				"type":        "runbook",
				"service":     "svc-payments",
				"environment": "prod",
			},
			Metadata: map[string]any{
				"source":      p.cfg.Source,
				"runbook_url": "https://runbook.demo/payment-latency",
				"team":        "payments",
			},
		},
	}

	for _, runbook := range runbooks {
		p.plans[runbook.ID] = runbook
	}
}

func (p *Provider) seedReleaseChecklists(now time.Time) {
	checklists := []schema.OrchestrationPlan{
		{
			ID:    "plan-release-001",
			Title: "Production Release Checklist",
			Description: "Comprehensive checklist for production releases. " +
				"Includes pre-deploy checks, approval, deployment, testing, and monitoring.",
			Steps: []schema.OrchestrationStep{
				{
					ID:          "step-1",
					Title:       "Pre-deploy checks",
					Type:        "manual",
					Description: "Verify all tests pass, dependencies are available, and rollback plan is ready.",
				},
				{
					ID:    "step-2",
					Title: "Get approval",
					Type:  "manual",
					Description: "Obtain approval from release manager and stakeholders. " +
						"Confirm maintenance window.",
					DependsOn: []string{"step-1"},
				},
				{
					ID:    "step-3",
					Title: "Deploy to production",
					Type:  "manual",
					Description: "Execute deployment to production environment. " +
						"Monitor deployment progress.",
					DependsOn: []string{"step-2"},
				},
				{
					ID:    "step-4a",
					Title: "Run smoke tests",
					Type:  "manual",
					Description: "Execute smoke tests to verify basic functionality. " +
						"Check critical user flows.",
					DependsOn: []string{"step-3"},
				},
				{
					ID:          "step-4b",
					Title:       "Run security scan",
					Type:        "manual",
					Description: "Perform automated vulnerability scan on new deployment.",
					DependsOn:   []string{"step-3"},
				},
				{
					ID:    "step-5",
					Title: "Monitor metrics",
					Type:  "manual",
					Description: "Monitor error rates, latency, and resource usage. " +
						"Watch for anomalies.",
					DependsOn: []string{"step-4a", "step-4b"},
				},
				{
					ID:    "step-6",
					Title: "Decide on rollback",
					Type:  "manual",
					Description: "Assess if rollback is needed based on metrics and errors. " +
						"Proceed or rollback.",
					DependsOn: []string{"step-5"},
				},
				{
					ID:    "step-7",
					Title: "Announce release",
					Type:  "manual",
					Description: "Post announcement about successful release. " +
						"Update status page.",
					DependsOn: []string{"step-6"},
				},
				{
					ID:    "step-8",
					Title: "Close release ticket",
					Type:  "manual",
					Description: "Close release ticket and document any issues. " +
						"Schedule postmortem if needed.",
					DependsOn: []string{"step-7"},
				},
			},
			URL:     "https://runbook.demo/checklists/prod-release",
			Version: "1.0",
			Tags: map[string]string{
				"type":        "release-checklist",
				"environment": "prod",
			},
			Metadata: map[string]any{
				"source": p.cfg.Source,
				"team":   "release-engineering",
			},
		},
		{
			ID:    "plan-release-002",
			Title: "Canary Deployment Checklist",
			Description: "Checklist for canary deployments with gradual rollout. " +
				"Includes staged rollout and monitoring at each stage.",
			Steps: []schema.OrchestrationStep{
				{
					ID:    "step-1",
					Title: "Deploy to 5% of traffic",
					Type:  "manual",
					Description: "Deploy new version to 5% of production traffic. " +
						"Use traffic splitting.",
				},
				{
					ID:    "step-2",
					Title: "Monitor 5% canary",
					Type:  "manual",
					Description: "Monitor error rates and latency for canary traffic. " +
						"Watch for 10 minutes.",
					DependsOn: []string{"step-1"},
				},
				{
					ID:          "step-3",
					Title:       "Deploy to 25% of traffic",
					Type:        "manual",
					Description: "Increase traffic to 25% if canary is healthy.",
					DependsOn:   []string{"step-2"},
				},
				{
					ID:    "step-4",
					Title: "Monitor 25% canary",
					Type:  "manual",
					Description: "Monitor error rates and latency for 25% traffic. " +
						"Watch for 10 minutes.",
					DependsOn: []string{"step-3"},
				},
				{
					ID:          "step-5",
					Title:       "Deploy to 100% of traffic",
					Type:        "manual",
					Description: "Roll out to all traffic if 25% canary is healthy.",
					DependsOn:   []string{"step-4"},
				},
				{
					ID:    "step-6",
					Title: "Verify full rollout",
					Type:  "manual",
					Description: "Confirm all traffic is on new version. " +
						"Monitor for 30 minutes.",
					DependsOn: []string{"step-5"},
				},
			},
			URL:     "https://runbook.demo/checklists/canary-deploy",
			Version: "1.0",
			Tags: map[string]string{
				"type":        "release-checklist",
				"environment": "prod",
			},
			Metadata: map[string]any{
				"source": p.cfg.Source,
				"team":   "release-engineering",
			},
		},
		{
			ID:    "plan-release-003",
			Title: "Rollback Checklist",
			Description: "Procedure for rolling back a failed deployment. " +
				"Includes assessment, revert, verification, and communication.",
			Steps: []schema.OrchestrationStep{
				{
					ID:    "step-1",
					Title: "Assess failure",
					Type:  "manual",
					Description: "Analyze error logs and metrics to understand failure. " +
						"Determine if rollback is necessary.",
				},
				{
					ID:    "step-2",
					Title: "Revert to previous version",
					Type:  "manual",
					Description: "Execute rollback to previous stable version. " +
						"Monitor deployment progress.",
					DependsOn: []string{"step-1"},
				},
				{
					ID:    "step-3",
					Title: "Verify rollback",
					Type:  "manual",
					Description: "Confirm previous version is running and healthy. " +
						"Run smoke tests.",
					DependsOn: []string{"step-2"},
				},
				{
					ID:    "step-4",
					Title: "Communicate rollback",
					Type:  "manual",
					Description: "Notify stakeholders about rollback. " +
						"Update status page.",
					DependsOn: []string{"step-3"},
				},
				{
					ID:    "step-5",
					Title: "Schedule postmortem",
					Type:  "manual",
					Description: "Schedule postmortem to review root cause. " +
						"Assign action items to prevent recurrence.",
					DependsOn: []string{"step-4"},
				},
			},
			URL:     "https://runbook.demo/checklists/rollback",
			Version: "1.0",
			Tags: map[string]string{
				"type":        "release-checklist",
				"environment": "prod",
			},
			Metadata: map[string]any{
				"source": p.cfg.Source,
				"team":   "release-engineering",
			},
		},
	}

	for _, checklist := range checklists {
		p.plans[checklist.ID] = checklist
	}
}

func (p *Provider) seedComplexFlows(now time.Time) {
	complexPlans := []schema.OrchestrationPlan{
		{
			ID:    "plan-complex-001",
			Title: "Multi-Service Feature Rollout",
			Description: "Orchestrated rollout of a new feature spanning multiple microservices. " +
				"Demonstrates diamond dependency pattern (1->2,3->4->7 ...).",
			Steps: []schema.OrchestrationStep{
				{
					ID:    "step-1",
					Title: "Prepare Shared Infrastructure",
					Type:  "manual",
					Description: "Provision necessary database schemas and shared message queues. " +
						"Ensure capacity for new feature load.",
				},
				{
					ID:    "step-2",
					Title: "Deploy Authorization Service",
					Type:  "manual",
					Description: "Deploy updated auth service with new scopes. " +
						"Verify backward compatibility.",
					DependsOn: []string{"step-1"},
				},
				{
					ID:    "step-3",
					Title: "Deploy Data Processing Service",
					Type:  "manual",
					Description: "Deploy new data processor consumers. " +
						"Start consuming from new topics.",
					DependsOn: []string{"step-1"},
				},
				{
					ID:    "step-4",
					Title: "Run Data Migration",
					Type:  "manual",
					Description: "Execute backfill migration script for existing users. " +
						"Validate data integrity.",
					DependsOn: []string{"step-3"},
				},
				{
					ID:    "step-5",
					Title: "Update API Gateway Policies",
					Type:  "manual",
					Description: "Update gateway routing and rate limits. " +
						"Enable new endpoints.",
					DependsOn: []string{"step-2"},
				},
				{
					ID:    "step-6",
					Title: "Refresh Client Tokens",
					Type:  "manual",
					Description: "Force refresh of client tokens to pick up new permissions. " +
						"Monitor auth error rates.",
					DependsOn: []string{"step-5"},
				},
				{
					ID:    "step-7",
					Title: "Enable Global Access",
					Type:  "manual",
					Description: "Flip feature flag to enable access for all users. " +
						"Send release notification.",
					DependsOn: []string{"step-4", "step-6"},
				},
			},
			URL:     "https://runbook.demo/rollouts/multi-service-feature",
			Version: "1.0",
			Tags:    map[string]string{"type": "complex-flow", "scenario": "rollout"},
			Metadata: map[string]any{
				"source": p.cfg.Source,
			},
		},
		{
			ID:    "plan-complex-002",
			Title: "Global Configuration Update",
			Description: "Concurrent update of configuration across all global regions. " +
				"Demonstrates fan-out/fan-in pattern.",
			Steps: []schema.OrchestrationStep{
				{
					ID:    "step-1",
					Title: "Initiate Global Update",
					Type:  "manual",
					Description: "Prepare configuration payload and version. " +
						"Acquire global lock.",
				},
				{
					ID:    "step-2",
					Title: "Update Region US-East",
					Type:  "manual",
					Description: "Apply configuration to us-east-1 and us-east-2. " +
						"Restart services if required.",
					DependsOn: []string{"step-1"},
				},
				{
					ID:    "step-3",
					Title: "Update Region US-West",
					Type:  "manual",
					Description: "Apply configuration to us-west-1 and us-west-2. " +
						"Restart services if required.",
					DependsOn: []string{"step-1"},
				},
				{
					ID:    "step-4",
					Title: "Update Region EU-Central",
					Type:  "manual",
					Description: "Apply configuration to eu-central-1. " +
						"Restart services if required.",
					DependsOn: []string{"step-1"},
				},
				{
					ID:    "step-5",
					Title: "Update Region AP-Southeast",
					Type:  "manual",
					Description: "Apply configuration to ap-southeast-1. " +
						"Restart services if required.",
					DependsOn: []string{"step-1"},
				},
				{
					ID:    "step-6",
					Title: "Verify Global Consistency",
					Type:  "manual",
					Description: "Check all regions report the new configuration version. " +
						"Release global lock.",
					DependsOn: []string{"step-2", "step-3", "step-4", "step-5"},
				},
			},
			URL:     "https://runbook.demo/ops/global-config-update",
			Version: "1.0",
			Tags:    map[string]string{"type": "complex-flow", "scenario": "ops"},
			Metadata: map[string]any{
				"source": p.cfg.Source,
			},
		},
		{
			ID:    "plan-complex-003",
			Title: "Legacy Monolith Upgrade",
			Description: "Strict sequence of steps required to upgrade legacy monolith. " +
				"Demonstrates long chain dependency.",
			Steps: []schema.OrchestrationStep{
				{ID: "step-1", Title: "Notify Maintenance Window", Type: "manual", Description: "Send email to internal stakeholders."},
				{ID: "step-2", Title: "Stop Background Jobs", Type: "manual", Description: "Pause all cron jobs and workers.", DependsOn: []string{"step-1"}},
				{ID: "step-3", Title: "Perform Full Backup", Type: "manual", Description: "Take snapshot of database and file storage.", DependsOn: []string{"step-2"}},
				{ID: "step-4", Title: "Enable Maintenance Mode", Type: "manual", Description: "Redirect traffic to maintenance page.", DependsOn: []string{"step-3"}},
				{ID: "step-5", Title: "Apply Database Patches", Type: "manual", Description: "Run SQL scripts for schema updates.", DependsOn: []string{"step-4"}},
				{ID: "step-6", Title: "Upgrade Application Binaries", Type: "manual", Description: "Replace executable on all nodes.", DependsOn: []string{"step-5"}},
				{ID: "step-7", Title: "Cold Restart", Type: "manual", Description: "Start application process.", DependsOn: []string{"step-6"}},
				{ID: "step-8", Title: "Verify Internal Health", Type: "manual", Description: "Check /health endpoint and logs.", DependsOn: []string{"step-7"}},
				{ID: "step-9", Title: "Disable Maintenance Mode", Type: "manual", Description: "Restore user traffic.", DependsOn: []string{"step-8"}},
				{ID: "step-10", Title: "Resume Background Jobs", Type: "manual", Description: "Unpause workers and verify processing.", DependsOn: []string{"step-9"}},
			},
			URL:     "https://runbook.demo/maintenance/monolith-upgrade",
			Version: "1.0",
			Tags:    map[string]string{"type": "complex-flow", "scenario": "maintenance"},
			Metadata: map[string]any{
				"source": p.cfg.Source,
			},
		},
		{
			ID:    "plan-complex-004",
			Title: "Full Stack Deployment",
			Description: "Parallel coordinated deployment of Mobile and Web stacks. " +
				"Demonstrates parallel tracks converging.",
			Steps: []schema.OrchestrationStep{
				{
					ID:          "step-1",
					Title:       "Mobile: Build iOS App",
					Type:        "manual",
					Description: "Compile Swift code and sign archive.",
				},
				{
					ID:          "step-2",
					Title:       "Mobile: Run UI Tests",
					Type:        "manual",
					Description: "Execute XCTest suite on simulators.",
					DependsOn:   []string{"step-1"},
				},
				{
					ID:          "step-3",
					Title:       "Mobile: Submit to App Store",
					Type:        "manual",
					Description: "Upload binary to TestFlight for review.",
					DependsOn:   []string{"step-2"},
				},
				{
					ID:          "step-4",
					Title:       "Web: Build Frontend",
					Type:        "manual",
					Description: "Run webpack build and optimize assets.",
				},
				{
					ID:          "step-5",
					Title:       "Web: Deploy to S3",
					Type:        "manual",
					Description: "Upload static assets to hosting bucket.",
					DependsOn:   []string{"step-4"},
				},
				{
					ID:          "step-6",
					Title:       "Web: Invalidate CDN",
					Type:        "manual",
					Description: "Purge CloudFront cache.",
					DependsOn:   []string{"step-5"},
				},
				{
					ID:          "step-7",
					Title:       "Release Announcement",
					Type:        "manual",
					Description: "Coordinate blog post and social media blast.",
					DependsOn:   []string{"step-3", "step-6"},
				},
			},
			URL:     "https://runbook.demo/releases/full-stack",
			Version: "1.0",
			Tags:    map[string]string{"type": "complex-flow", "scenario": "release"},
			Metadata: map[string]any{
				"source": p.cfg.Source,
			},
		},
		{
			ID:    "plan-complex-005",
			Title: "Data Center Migration (Complex DAG)",
			Description: "Simulates a large-scale DC migration with deep branching. " +
				"Start -> DB/Compute Branches -> DB splits to 3 tasks, Compute splits to 2 tasks -> Convergence.",
			Steps: []schema.OrchestrationStep{
				// Root
				{ID: "step-1", Title: "Initialize Migration", Type: "manual", Description: "Approve migration plan and notify users."},

				// Branch 1: Database
				{ID: "step-2", Title: "Prepare Database", Type: "manual", DependsOn: []string{"step-1"}, Description: "Allocate new DB resources."},
				{ID: "step-4", Title: "Backup Primary DB", Type: "manual", DependsOn: []string{"step-2"}, Description: "Take full snapshot."},
				{ID: "step-5", Title: "Provision Storage", Type: "manual", DependsOn: []string{"step-2"}, Description: "Setup high-performance and archival storage."},
				{ID: "step-6", Title: "Configure Replication", Type: "manual", DependsOn: []string{"step-2"}, Description: "Setup async replication to new region."},

				// Branch 2: Compute
				{ID: "step-3", Title: "Prepare Compute", Type: "manual", DependsOn: []string{"step-1"}, Description: "Reserve instance capacity."},
				{ID: "step-7", Title: "Deploy Control Plane", Type: "manual", DependsOn: []string{"step-3"}, Description: "Bootstrap Kubernetes masters."},
				{ID: "step-8", Title: "Deploy Worker Nodes", Type: "manual", DependsOn: []string{"step-3"}, Description: "Provision autoscaling node groups."},

				// Merge / Convergence
				{ID: "step-9", Title: "Verify Infrastructure", Type: "manual", DependsOn: []string{"step-4", "step-5", "step-6", "step-7", "step-8"}, Description: "Run integration tests on new environment."},
				{ID: "step-10", Title: "Switch Traffic", Type: "manual", DependsOn: []string{"step-9"}, Description: "Update global DNS to point to new DC."},
			},
			URL:     "https://runbook.demo/migrations/dc-move",
			Version: "1.0",
			Tags:    map[string]string{"type": "complex-flow", "scenario": "migration"},
			Metadata: map[string]any{
				"source": p.cfg.Source,
			},
		},
		{
			ID:    "plan-complex-006",
			Title: "Region Evacuation Protocol (Extreme DAG)",
			Description: "Full-scale region evacuation scenario. Features 3 parallel tracks (Data, Infra, Traffic), " +
				"cross-track dependencies (Restore needs Backup+Infra), and multiple convergence points.",
			Steps: []schema.OrchestrationStep{
				// Root
				{ID: "s1-init", Title: "Initialize Evacuation", Type: "manual", Description: "Declare incident and trigger evacuation protocol."},

				// Track A: State Preservation
				{ID: "s2-block-writes", Title: "Block Global Writes", Type: "manual", DependsOn: []string{"s1-init"}, Description: "Set applications to read-only mode."},
				{ID: "s3-backup-shard-1", Title: "Backup DB Shard 1", Type: "manual", DependsOn: []string{"s2-block-writes"}, Description: "Trigger immediate snapshot for Shard 1."},
				{ID: "s4-backup-shard-2", Title: "Backup DB Shard 2", Type: "manual", DependsOn: []string{"s2-block-writes"}, Description: "Trigger immediate snapshot for Shard 2."},
				{ID: "s5-snapshot-vols", Title: "Snapshot Block Volumes", Type: "manual", DependsOn: []string{"s2-block-writes"}, Description: "Snapshot EBS volumes for stateful sets."},

				// Track B: Traffic Management
				{ID: "s6-drain-traffic", Title: "Drain Regional Traffic", Type: "manual", DependsOn: []string{"s1-init"}, Description: "Lower weight in global load balancer."},
				{ID: "s7-update-cdn", Title: "Update CDN Origins", Type: "manual", DependsOn: []string{"s6-drain-traffic"}, Description: "Point CDN to fallback region."},

				// Track C: New Infrastructure Provisioning
				{ID: "s8-provision-vpc", Title: "Provision Disaster Recovery VPC", Type: "manual", DependsOn: []string{"s1-init"}, Description: "Terraform apply new VPC."},
				{ID: "s9-provision-rds", Title: "Provision New RDS Instances", Type: "manual", DependsOn: []string{"s8-provision-vpc"}, Description: "Create fresh DB instances in DR region."},
				{ID: "s10-provision-k8s", Title: "Provision K8s Cluster", Type: "manual", DependsOn: []string{"s8-provision-vpc"}, Description: "Boot up EKS cluster."},

				// Cross-Dependency Merge: Data Restore (Needs Backup + New Infra)
				{ID: "s11-restore-shard-1", Title: "Restore Shard 1 to DR", Type: "manual", DependsOn: []string{"s3-backup-shard-1", "s9-provision-rds"}, Description: "Restore snapshot 1 to new RDS."},
				{ID: "s12-restore-shard-2", Title: "Restore Shard 2 to DR", Type: "manual", DependsOn: []string{"s4-backup-shard-2", "s9-provision-rds"}, Description: "Restore snapshot 2 to new RDS."},

				// App Convergence (Needs K8s + Data Restored)
				{ID: "s13-deploy-app", Title: "Deploy Application Stack", Type: "manual", DependsOn: []string{"s10-provision-k8s", "s11-restore-shard-1", "s12-restore-shard-2"}, Description: "Helm install all microservices."},

				// Final Validation & Switch (Needs App + Traffic Prep + Volume Snapshots verified)
				{ID: "s14-verify-systems", Title: "Verify System Integrity", Type: "manual", DependsOn: []string{"s13-deploy-app", "s5-snapshot-vols"}, Description: "Run end-to-end smoke tests."},
				{ID: "s15-switch-dns", Title: "Global DNS Switchover", Type: "manual", DependsOn: []string{"s14-verify-systems", "s7-update-cdn"}, Description: "Update Route53 to point to DR region as primary."},
			},
			URL:     "https://runbook.demo/dr/region-evacuation",
			Version: "1.0",
			Tags:    map[string]string{"type": "complex-flow", "scenario": "dr"},
			Metadata: map[string]any{
				"source": p.cfg.Source,
			},
		},
	}

	for _, plan := range complexPlans {
		p.plans[plan.ID] = plan
	}
}

func (p *Provider) seedRuns(now time.Time) {
	runs := []schema.OrchestrationRun{
		{
			ID:     "run-001",
			PlanID: "plan-playbook-001",
			Status: "blocked",
			Scope: schema.QueryScope{
				Service:     "database",
				Environment: "prod",
			},
			Steps: []schema.OrchestrationStepState{
				{StepID: "step-1", Status: "succeeded", UpdatedAt: &now},
				{StepID: "step-2", Status: "succeeded", UpdatedAt: &now},
				{StepID: "step-3", Status: "ready", UpdatedAt: &now},
				{StepID: "step-4", Status: "pending", UpdatedAt: &now},
				{StepID: "step-5", Status: "pending", UpdatedAt: &now},
				{StepID: "step-6", Status: "pending", UpdatedAt: &now},
			},
			CreatedAt: now.Add(-30 * time.Minute),
			UpdatedAt: now.Add(-5 * time.Minute),
			Metadata: map[string]any{
				"source": p.cfg.Source,
			},
		},
		{
			ID:     "run-002",
			PlanID: "plan-release-001",
			Status: "running",
			Scope: schema.QueryScope{
				Environment: "prod",
			},
			Steps: []schema.OrchestrationStepState{
				{StepID: "step-1", Status: "succeeded", UpdatedAt: &now},
				{StepID: "step-2", Status: "succeeded", UpdatedAt: &now},
				{StepID: "step-3", Status: "running", UpdatedAt: &now},
				{StepID: "step-4", Status: "pending", UpdatedAt: &now},
				{StepID: "step-5", Status: "pending", UpdatedAt: &now},
				{StepID: "step-6", Status: "pending", UpdatedAt: &now},
				{StepID: "step-7", Status: "pending", UpdatedAt: &now},
				{StepID: "step-8", Status: "pending", UpdatedAt: &now},
			},
			CreatedAt: now.Add(-15 * time.Minute),
			UpdatedAt: now.Add(-2 * time.Minute),
			Metadata: map[string]any{
				"source": p.cfg.Source,
			},
		},
		{
			ID:     "run-003",
			PlanID: "plan-runbook-002",
			Status: "completed",
			Scope: schema.QueryScope{
				Environment: "prod",
			},
			Steps: []schema.OrchestrationStepState{
				{StepID: "step-1", Status: "succeeded", UpdatedAt: &now},
				{StepID: "step-2", Status: "succeeded", UpdatedAt: &now},
				{StepID: "step-3", Status: "succeeded", UpdatedAt: &now},
				{StepID: "step-4", Status: "succeeded", UpdatedAt: &now},
			},
			CreatedAt: now.Add(-2 * time.Hour),
			UpdatedAt: now.Add(-30 * time.Minute),
			Metadata: map[string]any{
				"source": p.cfg.Source,
			},
		},
		{
			ID:     "run-scenario-001",
			PlanID: "plan-playbook-001",
			Status: "running",
			Scope: schema.QueryScope{
				Service:     "database",
				Environment: "prod",
			},
			Steps: []schema.OrchestrationStepState{
				{StepID: "step-1", Status: "succeeded", UpdatedAt: &now},
				{StepID: "step-2", Status: "running", UpdatedAt: &now},
				{StepID: "step-3", Status: "pending", UpdatedAt: &now},
				{StepID: "step-4", Status: "pending", UpdatedAt: &now},
				{StepID: "step-5", Status: "pending", UpdatedAt: &now},
				{StepID: "step-6", Status: "pending", UpdatedAt: &now},
			},
			CreatedAt: now.Add(-10 * time.Minute),
			UpdatedAt: now.Add(-1 * time.Minute),
			Metadata: map[string]any{
				"source":      p.cfg.Source,
				"scenario_id": "active-incident-response",
				"is_scenario": true,
			},
		},
	}

	for _, run := range runs {
		if plan, ok := p.plans[run.PlanID]; ok {
			planClone := clonePlan(plan)
			run.Plan = &planClone
		}
		p.runs[run.ID] = run
	}
}
