import React, {useCallback, useEffect, useRef, useState} from 'react';
import {createPortal} from 'react-dom';
import PropTypes from 'prop-types';
import {Button, Icon, MenuItem, PopoverMenu, useOutsideClose} from '@mattermost/compass-ui';
import ChevronDownIcon from '@mattermost/compass-icons/components/chevron-down';

const MENU_WIDTH = 240;
const MENU_ITEM_HEIGHT = 40;
const MENU_PADDING = 16;
const MENU_GAP = 4;

// The menu renders in a portal with fixed coordinates because the sidebar scroll area clips its children.
function getMenuStyle(trigger, itemCount) {
    const rect = trigger.getBoundingClientRect();
    const style = {position: 'fixed', zIndex: 'var(--z-tooltip)'};

    const menuHeight = (itemCount * MENU_ITEM_HEIGHT) + MENU_PADDING;
    if ((rect.bottom + MENU_GAP + menuHeight) > window.innerHeight && rect.top > menuHeight) {
        style.bottom = (window.innerHeight - rect.top) + MENU_GAP;
    } else {
        style.top = rect.bottom + MENU_GAP;
    }

    if (rect.left + MENU_WIDTH > window.innerWidth) {
        style.right = Math.max(window.innerWidth - rect.right, 0);
    } else {
        style.left = rect.left;
    }

    return style;
}

function DebugMenu({title, items}) {
    const triggerRef = useRef(null);
    const menuRef = useRef(null);
    const [open, setOpen] = useState(false);
    const [mounted, setMounted] = useState(false);
    const [style, setStyle] = useState(null);

    const close = useCallback(() => setOpen(false), []);

    useOutsideClose(triggerRef, open, close, menuRef);

    useEffect(() => {
        const onKeyDown = (e) => {
            if (e.key === 'Escape') {
                setOpen(false);
                triggerRef.current?.focus();
            }
        };

        if (open) {
            document.addEventListener('keydown', onKeyDown);
            window.addEventListener('resize', close);
            window.addEventListener('scroll', close, true);
        }

        return () => {
            document.removeEventListener('keydown', onKeyDown);
            window.removeEventListener('resize', close);
            window.removeEventListener('scroll', close, true);
        };
    }, [open, close]);

    useEffect(() => {
        if (open && menuRef.current) {
            menuRef.current.querySelector('[role="menuitem"]')?.focus();
        }
    }, [open, mounted]);

    const toggle = () => {
        if (!open && triggerRef.current) {
            setStyle(getMenuStyle(triggerRef.current, items.length));
            setMounted(true);
        }
        setOpen(!open);
    };

    const onMenuKeyDown = (e) => {
        const entries = Array.from(menuRef.current?.querySelectorAll('[role="menuitem"]') || []);
        const current = entries.indexOf(document.activeElement);
        let next = -1;
        if (e.key === 'ArrowDown') {
            next = (current + 1) % entries.length;
        } else if (e.key === 'ArrowUp') {
            next = ((current - 1) + entries.length) % entries.length;
        } else if (e.key === 'Home') {
            next = 0;
        } else if (e.key === 'End') {
            next = entries.length - 1;
        }
        if (next >= 0) {
            e.preventDefault();
            entries[next].focus();
        }
    };

    return (
        <>
            <Button
                ref={triggerRef}
                type='button'
                emphasis='tertiary'
                size='small'
                aria-haspopup='menu'
                aria-expanded={open}
                trailingIcon={<Icon glyph={<ChevronDownIcon/>}/>}
                onClick={toggle}
            >
                {title}
            </Button>
            {mounted && createPortal(
                <div
                    ref={menuRef}
                    style={style}
                    onKeyDown={onMenuKeyDown}
                >
                    <PopoverMenu
                        role='menu'
                        open={open}
                        onExited={() => setMounted(false)}
                    >
                        {items.map((item) => (
                            <MenuItem
                                key={item.key}
                                role='menuitem'
                                label={item.label}
                                leadingElement={false}
                                onClick={() => {
                                    close();
                                    triggerRef.current?.focus();
                                    item.onClick();
                                }}
                            />
                        ))}
                    </PopoverMenu>
                </div>,
                document.body,
            )}
        </>
    );
}

DebugMenu.propTypes = {
    title: PropTypes.string.isRequired,
    items: PropTypes.arrayOf(PropTypes.shape({
        key: PropTypes.string.isRequired,
        label: PropTypes.string.isRequired,
        onClick: PropTypes.func.isRequired,
    })).isRequired,
};

export default DebugMenu;
