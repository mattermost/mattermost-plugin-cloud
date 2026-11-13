import React, {useEffect, useMemo, useState} from 'react';
import PropTypes from 'prop-types';
import {Button, Checkbox, Icon, Scrollbar, SectionNotice, Select, TextInput} from '@mattermost/compass-ui';
import LockOutlineIcon from '@mattermost/compass-icons/components/lock-outline';

import EnvEditor, {envRowsToMap, useEnvRows} from './env_editor';
import {getVersionLabel} from './installation_utils';

const NAME_PATTERN = /^[a-zA-Z0-9-]+$/;

function requiredLabel(text) {
    return (
        <>
            {text}
            <span aria-hidden='true'>{' *'}</span>
        </>
    );
}

function toOptions(values, current) {
    const list = [...(values || [])];
    if (current && !list.includes(current)) {
        list.unshift(current);
    }
    return list.map((value) => ({value, label: value}));
}

function parseKeyList(text) {
    return text.split(',').map((key) => key.trim()).filter(Boolean);
}

function initialValues(mode, installation, options) {
    if (mode === 'edit') {
        return {
            name: installation.Name,
            version: getVersionLabel(installation),
            size: installation.Size || '',
            license: '',
            image: installation.Image || '',
            affinity: installation.Affinity || '',
            database: installation.Database || '',
            filestore: installation.Filestore || '',
            testData: Boolean(installation.TestData),
        };
    }

    return {
        name: '',
        version: options.DefaultVersion || 'latest',
        size: options.DefaultSize || '',
        license: options.DefaultLicense || '',
        image: options.DefaultImage || '',
        affinity: options.DefaultAffinity || '',
        database: options.DefaultDatabase || '',
        filestore: options.DefaultFilestore || '',
        testData: false,
    };
}

// Builds the update request. Only fields that differ from the installation are sent.
export function buildUpdatePayload(installation, values, envRows, clearEnvText) {
    const payload = {installation_id: installation.ID};
    const version = values.version.trim();

    if (version && version !== getVersionLabel(installation)) {
        payload.version = version;
    }
    if (values.size && values.size !== installation.Size) {
        payload.size = values.size;
    }
    if (values.image && values.image !== installation.Image) {
        payload.image = values.image;
    }
    if (values.license) {
        payload.license = values.license;
    }

    const setEnv = envRowsToMap(envRows);
    if (Object.keys(setEnv).length > 0) {
        payload.set_env = setEnv;
    }
    const clearEnv = parseKeyList(clearEnvText);
    if (clearEnv.length > 0) {
        payload.clear_env = clearEnv;
    }

    return payload;
}

export function buildCreatePayload(values, envRows) {
    return {
        name: values.name.trim(),
        version: values.version.trim(),
        size: values.size,
        license: values.license,
        image: values.image,
        affinity: values.affinity,
        database: values.database,
        filestore: values.filestore,
        test_data: values.testData,
        env: envRowsToMap(envRows),
    };
}

export default function InstallationForm({mode, installation, options, onSubmit, onCancel, onDirtyChange}) {
    const editing = mode === 'edit';
    const config = options || {};
    const newEnvRow = useEnvRows();

    const [initial] = useState(() => initialValues(mode, installation, config));
    const [values, setValues] = useState(initial);
    const [nameTouched, setNameTouched] = useState(false);
    const [envRows, setEnvRows] = useState([]);
    const [clearEnvText, setClearEnvText] = useState('');
    const [busy, setBusy] = useState(false);
    const [error, setError] = useState('');

    const setValue = (field) => (value) => setValues((current) => ({...current, [field]: value}));

    const selectOptions = useMemo(() => ({
        sizes: toOptions(config.Sizes, values.size),
        licenses: toOptions(config.Licenses),
        images: toOptions(config.Images, values.image),
        affinities: toOptions(config.Affinities, values.affinity),
        databases: toOptions(config.Databases, values.database),
        filestores: toOptions(config.Filestores, values.filestore),
    }), [config, values.size, values.image, values.affinity, values.database, values.filestore]);

    const nameInvalid = !editing && values.name !== '' && !NAME_PATTERN.test(values.name);
    const showNameError = nameInvalid && nameTouched;
    const envInvalid = envRows.some((row) => row.value !== '' && row.key.trim() === '');

    let hasChanges;
    if (editing) {
        hasChanges = Object.keys(buildUpdatePayload(installation, values, envRows, clearEnvText)).length > 1;
    } else {
        hasChanges = values.name.trim() !== '' && values.version.trim() !== '';
    }
    const dirty = Object.keys(initial).some((field) => values[field] !== initial[field]) || envRows.length > 0 || clearEnvText !== '';

    useEffect(() => {
        if (onDirtyChange) {
            onDirtyChange(dirty);
        }
    }, [dirty, onDirtyChange]);

    useEffect(() => () => {
        if (onDirtyChange) {
            onDirtyChange(false);
        }
    }, [onDirtyChange]);

    const canSubmit = hasChanges && !nameInvalid && !envInvalid && !busy;

    const submit = async (e) => {
        e.preventDefault();
        if (!canSubmit) {
            return;
        }

        setBusy(true);
        setError('');
        const payload = editing ? buildUpdatePayload(installation, values, envRows, clearEnvText) : buildCreatePayload(values, envRows);
        const result = await onSubmit(payload);
        if (result && result.error) {
            setError(result.error);
            setBusy(false);
        }
    };

    const lockedIcon = <Icon glyph={<LockOutlineIcon/>}/>;

    return (
        <form
            className='CloudRHS__form'
            onSubmit={submit}
            noValidate={true}
        >
            <div className='CloudRHS__formScroll'>
                <Scrollbar style={{height: '100%'}}>
                    <div className='CloudRHS__formBody'>
                        {error && (
                            <SectionNotice
                                type='danger'
                                title={editing ? 'Could not apply the changes' : 'Could not create the server'}
                                description={error}
                            />
                        )}

                        {editing && (
                            <p className='CloudRHS__hint'>
                                {'Locked fields cannot change after the server is created.'}
                            </p>
                        )}

                        <fieldset
                            className='CloudRHS__fieldset'
                            disabled={busy}
                        >
                            <legend>{'General'}</legend>
                            <TextInput
                                label={editing ? 'Name' : requiredLabel('Name')}
                                required={!editing}
                                value={values.name}
                                disabled={editing}
                                trailingIcon={editing ? lockedIcon : null}
                                invalid={showNameError}
                                message={showNameError ? 'Use only letters, numbers, and hyphens.' : ''}
                                onChange={(e) => setValue('name')(e.target.value)}
                                onBlur={() => setNameTouched(true)}
                            />
                            <TextInput
                                label={requiredLabel('Version')}
                                required={true}
                                value={values.version}
                                onChange={(e) => setValue('version')(e.target.value)}
                            />
                        </fieldset>

                        <fieldset
                            className='CloudRHS__fieldset'
                            disabled={busy}
                        >
                            <legend>{'Configuration'}</legend>
                            <Select
                                label='Size'
                                options={selectOptions.sizes}
                                value={values.size}
                                onChange={setValue('size')}
                            />
                            <Select
                                label='License'
                                placeholder={editing ? 'Keep current license' : 'Select a license'}
                                options={selectOptions.licenses}
                                value={values.license}
                                onChange={setValue('license')}
                            />
                            <Select
                                label='Image'
                                options={selectOptions.images}
                                value={values.image}
                                onChange={setValue('image')}
                            />
                        </fieldset>

                        <fieldset
                            className='CloudRHS__fieldset'
                            disabled={busy}
                        >
                            <legend>{'Infrastructure'}</legend>
                            <Select
                                label='Affinity'
                                options={selectOptions.affinities}
                                value={values.affinity}
                                disabled={editing}
                                onChange={setValue('affinity')}
                            />
                            <Select
                                label='Database'
                                options={selectOptions.databases}
                                value={values.database}
                                disabled={editing}
                                onChange={setValue('database')}
                            />
                            <Select
                                label='Filestore'
                                options={selectOptions.filestores}
                                value={values.filestore}
                                disabled={editing}
                                onChange={setValue('filestore')}
                            />
                            <Checkbox
                                checked={values.testData}
                                disabled={editing}
                                onChange={(e) => setValue('testData')(e.target.checked)}
                            >
                                {'Pre-load test data'}
                            </Checkbox>
                        </fieldset>

                        <fieldset
                            className='CloudRHS__fieldset'
                            disabled={busy}
                        >
                            <legend>{'Environment variables'}</legend>
                            {editing && (
                                <p className='CloudRHS__hint'>
                                    {'Existing values are hidden. Variables listed here replace the current values.'}
                                </p>
                            )}
                            <EnvEditor
                                rows={envRows}
                                onChange={setEnvRows}
                                newRow={newEnvRow}
                                disabled={busy}
                            />
                            {editing && (
                                <TextInput
                                    label='Names to remove'
                                    message='Separate names with commas.'
                                    messageTone='info'
                                    value={clearEnvText}
                                    onChange={(e) => setClearEnvText(e.target.value)}
                                />
                            )}
                        </fieldset>
                    </div>
                </Scrollbar>
            </div>

            <div className='CloudRHS__formActions'>
                <Button
                    type='submit'
                    emphasis='primary'
                    loading={busy}
                    disabled={!canSubmit}
                >
                    {editing ? 'Apply' : 'Create'}
                </Button>
                <Button
                    type='button'
                    emphasis='tertiary'
                    disabled={busy}
                    onClick={onCancel}
                >
                    {'Cancel'}
                </Button>
            </div>
        </form>
    );
}

InstallationForm.propTypes = {
    mode: PropTypes.oneOf(['create', 'edit']).isRequired,
    installation: PropTypes.object,
    options: PropTypes.object,
    onSubmit: PropTypes.func.isRequired,
    onCancel: PropTypes.func.isRequired,
    onDirtyChange: PropTypes.func,
};
