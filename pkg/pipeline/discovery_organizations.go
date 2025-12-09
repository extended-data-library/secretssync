package pipeline

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

// discoverFromOrganizations discovers accounts from AWS Organizations
func (d *DiscoveryService) discoverFromOrganizations(cfg *OrganizationsDiscovery) ([]AccountInfo, error) {
	l := log.WithFields(log.Fields{
		"action":    "discoverFromOrganizations",
		"ou":        cfg.OU,
		"recursive": cfg.Recursive,
	})
	l.Debug("Discovering accounts from Organizations")

	if !d.awsCtx.CanAccessOrganizations() {
		return nil, fmt.Errorf("no access to Organizations API from this execution context")
	}

	var accounts []AccountInfo

	// Discover by OU
	if cfg.OU != "" {
		if cfg.Recursive {
			// Recursive traversal of OU and all child OUs
			ouAccounts, err := d.listAccountsInOURecursive(cfg.OU)
			if err != nil {
				return nil, err
			}
			accounts = append(accounts, ouAccounts...)
		} else {
			// Direct children only
			ouAccounts, err := d.awsCtx.ListAccountsInOU(d.ctx, cfg.OU)
			if err != nil {
				return nil, err
			}
			accounts = append(accounts, ouAccounts...)
		}
	}

	// If no OU specified but tags are specified, list all accounts and filter
	if cfg.OU == "" && len(cfg.Tags) > 0 {
		allAccounts, err := d.awsCtx.ListOrganizationAccounts(d.ctx)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, allAccounts...)
	}

	// Filter by tags if specified
	if len(cfg.Tags) > 0 {
		accounts = filterAccountsByTags(accounts, cfg.Tags)
	}

	l.WithField("count", len(accounts)).Debug("Discovered accounts from Organizations")
	return accounts, nil
}

// listAccountsInOURecursive recursively lists accounts in an OU and all child OUs
func (d *DiscoveryService) listAccountsInOURecursive(ouID string) ([]AccountInfo, error) {
	var accounts []AccountInfo

	// Get accounts directly in this OU
	ouAccounts, err := d.awsCtx.ListAccountsInOU(d.ctx, ouID)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts in OU %s: %w", ouID, err)
	}
	accounts = append(accounts, ouAccounts...)

	// Get child OUs and recurse
	childOUs, err := d.awsCtx.ListChildOUs(d.ctx, ouID)
	if err != nil {
		// Log but continue - we might not have permission to list child OUs
		log.WithError(err).WithField("ou", ouID).Debug("Could not list child OUs")
		return accounts, nil
	}

	for _, childOU := range childOUs {
		childAccounts, err := d.listAccountsInOURecursive(childOU)
		if err != nil {
			log.WithError(err).WithField("childOU", childOU).Debug("Error recursing into child OU")
			continue
		}
		accounts = append(accounts, childAccounts...)
	}

	return accounts, nil
}
