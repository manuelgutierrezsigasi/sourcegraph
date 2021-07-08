import * as H from 'history'
import React, { ReactElement, useMemo } from 'react'

import { displayRepoName } from '@sourcegraph/shared/src/components/RepoFileLink'
import { ThemeProps } from '@sourcegraph/shared/src/theme'
import { useObservable } from '@sourcegraph/shared/src/util/useObservable'
import { DiffStat } from '@sourcegraph/web/src/components/diff/DiffStat'
import { PageHeader } from '@sourcegraph/wildcard'

import { BatchChangesIcon } from '../../../batches/icons'
import { Page } from '../../../components/Page'
import { PageTitle } from '../../../components/PageTitle'
import { Maybe, RepositoryFields, RepoBatchChangeStats } from '../../../graphql-operations'
import { queryExternalChangesetWithFileDiffs as _queryExternalChangesetWithFileDiffs } from '../detail/backend'
import { BatchChangeStatsTotalAction } from '../detail/BatchChangeStatsCard'
import {
    ChangesetStatusUnpublished,
    ChangesetStatusOpen,
    ChangesetStatusClosed,
    ChangesetStatusMerged,
} from '../detail/changesets/ChangesetStatusCell'

import {
    queryRepoBatchChanges as _queryRepoBatchChanges,
    queryRepoBatchChangeStats as _queryRepoBatchChangeStats,
} from './backend'
import { RepoBatchChanges } from './RepoBatchChanges'

interface BatchChangeRepoPageProps extends ThemeProps {
    history: H.History
    location: H.Location
    repo: RepositoryFields
    /** For testing only. */
    queryRepoBatchChangeStats?: typeof _queryRepoBatchChangeStats
    /** For testing only. */
    queryRepoBatchChanges?: typeof _queryRepoBatchChanges
    /** For testing only. */
    queryExternalChangesetWithFileDiffs?: typeof _queryExternalChangesetWithFileDiffs
}

export const BatchChangeRepoPage: React.FunctionComponent<BatchChangeRepoPageProps> = ({
    repo,
    queryRepoBatchChangeStats = _queryRepoBatchChangeStats,
    ...context
}) => {
    const repoDisplayName = displayRepoName(repo.name)

    const stats: Maybe<RepoBatchChangeStats> | undefined = useObservable(
        useMemo(() => queryRepoBatchChangeStats({ name: repo.name }), [queryRepoBatchChangeStats, repo.name])
    )
    const hasChangesets = stats?.changesetsStats.total

    return (
        <Page>
            <PageTitle title="Batch Changes" />
            <PageHeader path={[{ icon: BatchChangesIcon, text: 'Batch Changes' }]} headingElement="h1" />
            {hasChangesets && stats?.batchChangesDiffStat && stats?.changesetsStats && (
                <div className="d-flex align-items-center mt-4 mb-3">
                    <h2 className="mb-0 pb-1">{repoDisplayName}</h2>
                    <DiffStat className="d-flex flex-1 ml-2" expandedCounts={true} {...stats.batchChangesDiffStat} />
                    <StatsBar stats={stats.changesetsStats} />
                </div>
            )}
            {hasChangesets ? (
                <p>
                    Batch changes has created {stats?.changesetsStats.total} changesets on {repoDisplayName}
                </p>
            ) : (
                <div className="mb-3" />
            )}
            <RepoBatchChanges viewerCanAdminister={true} repo={repo} {...context} />
        </Page>
    )
}

const ACTION_CLASSNAMES = 'd-flex flex-column text-muted justify-content-center align-items-center mx-2'

// TODO: Generalize icon label type to accept strings
const element = (string: string): ReactElement => <span>{string}</span>

interface StatsBarProps {
    stats: RepoBatchChangeStats['changesetsStats']
}

const StatsBar: React.FunctionComponent<StatsBarProps> = ({ stats: { total, open, unpublished, closed, merged } }) => (
    <div className="d-flex flex-wrap align-items-center">
        <BatchChangeStatsTotalAction count={total} />
        <ChangesetStatusOpen className={ACTION_CLASSNAMES} label={element(`${open} Open`)} />
        <ChangesetStatusUnpublished className={ACTION_CLASSNAMES} label={element(`${unpublished} Unpublished`)} />
        <ChangesetStatusClosed className={ACTION_CLASSNAMES} label={element(`${closed} Closed`)} />
        <ChangesetStatusMerged className={ACTION_CLASSNAMES} label={element(`${merged} Merged`)} />
    </div>
)
