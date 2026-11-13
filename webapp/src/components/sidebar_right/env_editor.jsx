import React, {useRef} from 'react';
import PropTypes from 'prop-types';
import {Button, Icon, IconButton, TextInput} from '@mattermost/compass-ui';
import CloseIcon from '@mattermost/compass-icons/components/close';
import PlusIcon from '@mattermost/compass-icons/components/plus';

// Rows are {id, key, value}. Ids are stable so removing a row does not shift the inputs of other rows.
export function useEnvRows() {
    const nextID = useRef(0);
    const newRow = () => {
        nextID.current += 1;
        return {id: nextID.current, key: '', value: ''};
    };
    return newRow;
}

export function envRowsToMap(rows) {
    const env = {};
    rows.forEach((row) => {
        const key = row.key.trim();
        if (key) {
            env[key] = row.value;
        }
    });
    return env;
}

export default function EnvEditor({rows, onChange, newRow, disabled}) {
    const updateRow = (id, field, value) => onChange(rows.map((row) => (row.id === id ? {...row, [field]: value} : row)));
    const removeRow = (id) => onChange(rows.filter((row) => row.id !== id));

    return (
        <div className='CloudRHS__envEditor'>
            {rows.map((row, index) => (
                <div
                    key={row.id}
                    className='CloudRHS__envRow'
                >
                    <TextInput
                        size='small'
                        label='Name'
                        aria-label={`Variable name ${index + 1}`}
                        value={row.key}
                        disabled={disabled}
                        onChange={(e) => updateRow(row.id, 'key', e.target.value)}
                    />
                    <TextInput
                        size='small'
                        label='Value'
                        aria-label={`Variable value ${index + 1}`}
                        value={row.value}
                        disabled={disabled}
                        onChange={(e) => updateRow(row.id, 'value', e.target.value)}
                    />
                    <IconButton
                        size='small'
                        aria-label={`Remove variable ${index + 1}`}
                        icon={<Icon glyph={<CloseIcon/>}/>}
                        disabled={disabled}
                        onClick={() => removeRow(row.id)}
                    />
                </div>
            ))}
            <div>
                <Button
                    type='button'
                    size='small'
                    emphasis='tertiary'
                    leadingIcon={<Icon glyph={<PlusIcon/>}/>}
                    disabled={disabled}
                    onClick={() => onChange([...rows, newRow()])}
                >
                    {'Add variable'}
                </Button>
            </div>
        </div>
    );
}

EnvEditor.propTypes = {
    rows: PropTypes.arrayOf(PropTypes.shape({
        id: PropTypes.number.isRequired,
        key: PropTypes.string.isRequired,
        value: PropTypes.string.isRequired,
    })).isRequired,
    onChange: PropTypes.func.isRequired,
    newRow: PropTypes.func.isRequired,
    disabled: PropTypes.bool,
};
