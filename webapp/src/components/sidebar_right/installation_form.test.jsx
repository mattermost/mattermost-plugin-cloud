import React from 'react';
import {act, fireEvent, render, screen, waitFor} from '@testing-library/react';

import InstallationForm, {buildCreatePayload, buildUpdatePayload} from './installation_form';

const options = {
    Sizes: ['miniSingleton', 'miniHA'],
    Licenses: ['enterprise', 'professional'],
    Images: ['mattermost/mattermost-enterprise-edition'],
    Affinities: ['multitenant', 'isolated'],
    Databases: ['db-a'],
    Filestores: ['fs-a'],
    DefaultSize: 'miniSingleton',
    DefaultVersion: 'latest',
    DefaultLicense: 'enterprise',
    DefaultImage: 'mattermost/mattermost-enterprise-edition',
    DefaultAffinity: 'multitenant',
    DefaultDatabase: 'db-a',
    DefaultFilestore: 'fs-a',
};

const installation = {
    ID: 'install-id',
    Name: 'existing',
    Tag: '9.5.0',
    Version: 'sha256:abc',
    Size: 'miniSingleton',
    Image: 'mattermost/mattermost-enterprise-edition',
    Affinity: 'multitenant',
    Database: 'db-a',
    Filestore: 'fs-a',
    TestData: true,
};

describe('InstallationForm create', () => {
    it('submits the defaults with the entered name and environment variables', async () => {
        const onSubmit = jest.fn().mockResolvedValue({data: {}});
        render(
            <InstallationForm
                mode='create'
                options={options}
                onSubmit={onSubmit}
                onCancel={jest.fn()}
            />,
        );

        fireEvent.change(screen.getByLabelText(/^Name( \*)?$/), {target: {value: 'my-server'}});
        fireEvent.click(screen.getByRole('button', {name: 'Add variable'}));
        fireEvent.change(screen.getByLabelText('Variable name 1'), {target: {value: 'MM_FOO'}});
        fireEvent.change(screen.getByLabelText('Variable value 1'), {target: {value: 'bar'}});
        await act(async () => {
            fireEvent.click(screen.getByRole('button', {name: 'Create'}));
        });

        expect(onSubmit).toHaveBeenCalledWith({
            name: 'my-server',
            version: 'latest',
            size: 'miniSingleton',
            license: 'enterprise',
            image: 'mattermost/mattermost-enterprise-edition',
            affinity: 'multitenant',
            database: 'db-a',
            filestore: 'fs-a',
            test_data: false,
            env: {MM_FOO: 'bar'},
        });
    });

    it('blocks submit without a name or with an invalid name', () => {
        render(
            <InstallationForm
                mode='create'
                options={options}
                onSubmit={jest.fn()}
                onCancel={jest.fn()}
            />,
        );
        expect(screen.getByRole('button', {name: 'Create'})).toBeDisabled();

        fireEvent.change(screen.getByLabelText(/^Name( \*)?$/), {target: {value: 'bad name!'}});
        fireEvent.blur(screen.getByLabelText(/^Name( \*)?$/));

        expect(screen.getByText('Use only letters, numbers, and hyphens.')).toBeInTheDocument();
        expect(screen.getByRole('button', {name: 'Create'})).toBeDisabled();
    });

    it('blocks submit when a variable has a value but no name', () => {
        render(
            <InstallationForm
                mode='create'
                options={options}
                onSubmit={jest.fn()}
                onCancel={jest.fn()}
            />,
        );
        fireEvent.change(screen.getByLabelText(/^Name( \*)?$/), {target: {value: 'ok'}});
        fireEvent.click(screen.getByRole('button', {name: 'Add variable'}));
        fireEvent.change(screen.getByLabelText('Variable value 1'), {target: {value: 'bar'}});

        expect(screen.getByRole('button', {name: 'Create'})).toBeDisabled();

        fireEvent.click(screen.getByRole('button', {name: 'Remove variable 1'}));
        expect(screen.getByRole('button', {name: 'Create'})).toBeEnabled();
    });

    it('shows the server error and keeps the values', async () => {
        const onSubmit = jest.fn().mockResolvedValue({error: 'name already exists'});
        render(
            <InstallationForm
                mode='create'
                options={options}
                onSubmit={onSubmit}
                onCancel={jest.fn()}
            />,
        );
        fireEvent.change(screen.getByLabelText(/^Name( \*)?$/), {target: {value: 'dup'}});
        await act(async () => {
            fireEvent.click(screen.getByRole('button', {name: 'Create'}));
        });

        expect(screen.getByText('name already exists')).toBeInTheDocument();
        expect(screen.getByLabelText(/^Name( \*)?$/)).toHaveValue('dup');
        await waitFor(() => expect(screen.getByRole('button', {name: 'Create'})).toBeEnabled());
    });
});

describe('InstallationForm edit', () => {
    const renderEdit = (onSubmit = jest.fn().mockResolvedValue({data: {}})) => {
        render(
            <InstallationForm
                mode='edit'
                installation={installation}
                options={options}
                onSubmit={onSubmit}
                onCancel={jest.fn()}
            />,
        );
        return onSubmit;
    };

    it('locks the fields that cannot change', () => {
        renderEdit();

        expect(screen.getByLabelText(/^Name( \*)?$/)).toBeDisabled();
        expect(screen.getByRole('checkbox', {name: 'Pre-load test data'})).toBeDisabled();
        expect(screen.getByRole('checkbox', {name: 'Pre-load test data'})).toBeChecked();
        ['Affinity', 'Database', 'Filestore'].forEach((label) => {
            expect(screen.getByRole('combobox', {name: new RegExp(label)})).toBeDisabled();
        });
        expect(screen.getByLabelText(/^Version/)).toBeEnabled();
        expect(screen.getByRole('combobox', {name: /Size/})).toBeEnabled();
    });

    it('keeps Apply disabled until something changes', () => {
        renderEdit();
        const apply = screen.getByRole('button', {name: 'Apply'});
        expect(apply).toBeDisabled();

        fireEvent.change(screen.getByLabelText(/^Version/), {target: {value: '9.6.0'}});
        expect(apply).toBeEnabled();

        fireEvent.change(screen.getByLabelText(/^Version/), {target: {value: '9.5.0'}});
        expect(apply).toBeDisabled();
    });

    it('sends only the changed fields', async () => {
        const onSubmit = renderEdit();

        fireEvent.change(screen.getByLabelText(/^Version/), {target: {value: '9.6.0'}});
        fireEvent.change(screen.getByLabelText('Names to remove'), {target: {value: 'OLD_ONE, OLD_TWO'}});
        await act(async () => {
            fireEvent.click(screen.getByRole('button', {name: 'Apply'}));
        });

        expect(onSubmit).toHaveBeenCalledWith({
            installation_id: 'install-id',
            version: '9.6.0',
            clear_env: ['OLD_ONE', 'OLD_TWO'],
        });
    });
});

describe('payload builders', () => {
    const values = {
        name: ' Name ',
        version: ' 9.5.0 ',
        size: 'miniHA',
        license: '',
        image: 'img',
        affinity: 'multitenant',
        database: 'db',
        filestore: 'fs',
        testData: true,
    };

    it('trims create fields and drops blank environment names', () => {
        const payload = buildCreatePayload(values, [{id: 1, key: ' A ', value: '1'}, {id: 2, key: '', value: ''}]);

        expect(payload).toMatchObject({name: 'Name', version: '9.5.0', test_data: true, env: {A: '1'}});
    });

    it('treats a license choice as a change and an empty license as keep', () => {
        const unchanged = {...values, version: '9.5.0', size: 'miniSingleton', image: installation.Image, license: ''};
        expect(buildUpdatePayload(installation, unchanged, [], '')).toEqual({installation_id: 'install-id'});

        expect(buildUpdatePayload(installation, {...unchanged, license: 'professional'}, [], '')).toEqual({installation_id: 'install-id', license: 'professional'});
    });
});
