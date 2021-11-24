import React, { FunctionComponent } from 'react'

import { LsifIndexFields } from '../../../../graphql-operations'
import { CodeIntelUploadOrIndexCommit } from '../../shared/components/CodeIntelUploadOrIndexCommit'
import { CodeIntelUploadOrIndexRepository } from '../../shared/components/CodeIntelUploadOrIndexerRepository'
import { CodeIntelUploadOrIndexIndexer } from '../../shared/components/CodeIntelUploadOrIndexIndexer'
import { CodeIntelUploadOrIndexLastActivity } from '../../shared/components/CodeIntelUploadOrIndexLastActivity'
import { CodeIntelUploadOrIndexRoot } from '../../shared/components/CodeIntelUploadOrIndexRoot'

export interface CodeIntelIndexMetaProps {
    node: LsifIndexFields
    now?: () => Date
}

export const CodeIntelIndexMeta: FunctionComponent<CodeIntelIndexMetaProps> = ({ node, now }) => (
    <div className="card">
        <div className="card-body">
            <div className="card border-0">
                <div className="card-body">
                    <h3 className="card-title">
                        <CodeIntelUploadOrIndexRepository node={node} />
                    </h3>

                    <p className="card-subtitle mb-2 text-muted">
                        <CodeIntelUploadOrIndexLastActivity node={{ ...node, uploadedAt: null }} now={now} />
                    </p>

                    <p className="card-text">
                        Directory <CodeIntelUploadOrIndexRoot node={node} /> indexed at commit{' '}
                        <CodeIntelUploadOrIndexCommit node={node} /> by <CodeIntelUploadOrIndexIndexer node={node} />
                    </p>
                </div>
            </div>
        </div>
    </div>
)