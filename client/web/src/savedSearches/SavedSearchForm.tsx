import classNames from 'classnames'
import React, { useMemo, useState } from 'react'
import { Omit } from 'utility-types'

import { Form } from '@sourcegraph/branded/src/components/Form'
import { Link } from '@sourcegraph/shared/src/components/Link'
import { Scalars } from '@sourcegraph/shared/src/graphql-operations'
import { Container, PageHeader } from '@sourcegraph/wildcard'

import { AuthenticatedUser } from '../auth'
import { ErrorAlert } from '../components/alerts'
import { Badge } from '../components/Badge'
import { NamespaceProps } from '../namespaces'

import styles from './SavedSearchForm.module.scss'

export interface SavedQueryFields {
    id: Scalars['ID']
    description: string
    query: string
}

export interface SavedSearchFormProps extends NamespaceProps {
    authenticatedUser: AuthenticatedUser | null
    defaultValues?: Partial<SavedQueryFields>
    title?: string
    submitLabel: string
    onSubmit: (fields: Omit<SavedQueryFields, 'id'>) => void
    loading: boolean
    error?: any
}

export const SavedSearchForm: React.FunctionComponent<SavedSearchFormProps> = props => {
    const [values, setValues] = useState<Omit<SavedQueryFields, 'id'>>(() => ({
        description: props.defaultValues?.description || '',
        query: props.defaultValues?.query || '',
    }))

    /**
     * Returns an input change handler that updates the SavedQueryFields in the component's state
     *
     * @param key The key of saved query fields that a change of this input should update
     */
    const createInputChangeHandler = (
        key: keyof SavedQueryFields
    ): React.FormEventHandler<HTMLInputElement> => event => {
        const { value, checked, type } = event.currentTarget
        setValues(values => ({
            ...values,
            [key]: type === 'checkbox' ? checked : value,
        }))
    }

    const handleSubmit = (event: React.FormEvent<HTMLFormElement>): void => {
        event.preventDefault()
        props.onSubmit(values)
    }

    const { query, description, notify } = values

    return (
        <div className="saved-search-form">
            <PageHeader
                path={[{ text: props.title }]}
                headingElement="h2"
                description="Save your common searches."
                className="mb-3"
            />
            <Form onSubmit={handleSubmit}>
                <Container className="mb-3">
                    <div className="form-group">
                        <label className={styles.label} htmlFor="saved-search-form-input-description">
                            Description
                        </label>
                        <input
                            id="saved-search-form-input-description"
                            type="text"
                            name="description"
                            className="form-control test-saved-search-form-input-description"
                            placeholder="Description"
                            required={true}
                            value={description}
                            onChange={createInputChangeHandler('description')}
                        />
                    </div>
                    <div className="form-group">
                        <label className={styles.label} htmlFor="saved-search-form-input-query">
                            Query
                        </label>
                        <input
                            id="saved-search-form-input-query"
                            type="text"
                            name="query"
                            className="form-control test-saved-search-form-input-query"
                            placeholder="Query"
                            required={true}
                            value={query}
                            onChange={createInputChangeHandler('query')}
                        />
                    </div>
                </Container>

                <button
                    type="submit"
                    disabled={props.loading}
                    className={classNames(styles.submitButton, 'btn btn-primary test-saved-search-form-submit-button')}
                >
                    {props.submitLabel}
                </button>

                {props.error && !props.loading && <ErrorAlert className="mb-3" error={props.error} />}

                <Container className="d-flex p-3 align-items-start">
                    <Badge status="new" className="mr-3">
                        New
                    </Badge>
                    <span>
                        Watch for changes to your code and trigger email notifications, webhooks, and more with{' '}
                        <Link to="/code-monitoring">code monitoring →</Link>
                    </span>
                </Container>
            </Form>
        </div>
    )
}
