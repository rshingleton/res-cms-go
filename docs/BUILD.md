# ResCMS Build Documentation

## Overview

ResCMS uses Tailwind CSS for styling. The CSS is built automatically via a custom Mojolicious plugin that integrates the Tailwind CLI into the application lifecycle.

## Tailwind CSS Plugin

The project includes a custom Mojolicious plugin (`ResCMS::Plugin::Tailwind`) that handles CSS builds.

### How It Works

**File:** `lib/ResCMS/Plugin/Tailwind.pm`

```perl
package ResCMS::Plugin::Tailwind;
use Mojo::Base 'Mojolicious::Plugin', -signatures;
use File::stat;

sub register ($self, $app, $conf) {
  my $input  = $conf->{input}  // 'styles/app.css';
  my $output = $conf->{output} // 'public/css/app.css';

  my $build = sub {
    $app->log->info('Building Tailwind CSS...');
    local $ENV{BROWSERSLIST_IGNORE_OLD_DATA} = '1';
    my $build_output = `$bin -i $input -o $output --minify 2>&1`;
    # ... logging
  };

  # Build on startup
  $build->();

  # Rebuild if source or templates are newer than output
  $app->hook(after_dispatch => sub {
    my $c = shift;
    return unless -e $output;
    
    my $out_time = stat($output)->mtime;
    my $rebuild_needed = 0;

    # Check main CSS source
    if (-e $input && stat($input)->mtime > $out_time) {
      $app->log->info('Tailwind source updated, rebuilding...');
      $rebuild_needed = 1;
    }

    # Check templates
    if (!$rebuild_needed) {
      my @templates = glob('templates/**/*.ep');
      foreach my $file (@templates) {
        if (stat($file)->mtime > $out_time) {
          $app->log->info("Template $file updated, rebuilding CSS...");
          $rebuild_needed = 1;
          last;
        }
      }
    }

    $build->() if $rebuild_needed;
  });
```

### Features

- **Automatic build on startup** - CSS is built when the application starts
- **Intelligent Rebuilding** - Detects changes in both `styles/app.css` AND all `.ep` templates in the `templates/` directory.
- **Development mode only** - Plugin only loads when `$app->mode eq 'development'`
- **Minified output** - Uses `--minify` flag for production-ready CSS

## Asset Versioning

The application uses an `asset_version` helper to ensure browser cache is invalidated whenever assets are rebuilt.

```html
<link rel="stylesheet" href="<%= asset_version '/css/app.css' %>">
```

This appends the file's modification time as a query parameter (`?v=...`), ensuring that clients always download the latest version of the CSS.

## Configuration

### Plugin Registration

The plugin is registered in `lib/ResCMS.pm`:

```perl
# Load Tailwind CSS plugin (only in development mode)
if ($self->mode eq 'development') {
  $self->plugin('ResCMS::Plugin::Tailwind');
}
```

### Source Files

- **Input:** `styles/app.css` - Tailwind source with `@tailwind` directives
- **Output:** `public/css/app.css` - Compiled CSS

### Tailwind Configuration

```javascript
// tailwind.config.js
module.exports = {
  content: [
    './templates/**/*.{ep,html.ep}',
    './public/**/*.{html,htm}'
  ],
  theme: {
    extend: {}
  },
  plugins: [
    require('@tailwindcss/forms'),
    require('@tailwindcss/typography')
  ]
};
```

## Building CSS Manually

The standalone Tailwind CLI is in `bin/tailwindcss`. Rebuild manually:

```bash
# One-time build
bin/tailwindcss -i styles/app.css -o public/css/app.css --minify

# Watch mode
bin/tailwindcss -i styles/app.css -o public/css/app.css --watch
```

## morbo and Hot Reloading

The application uses `morbo` for development, which provides:

- **Perl file watching** - `.pm` and `.ep` files trigger reloads
- **Plugin-based CSS rebuilding** - The Tailwind plugin has its own file watcher that checks `styles/app.css` modification time on each request
- **Automatic regeneration** - When you save changes to Tailwind source, CSS is regenerated on the next page load

### Workflow

1. Start the server: `morbo script/rescms`
2. Edit templates (`.ep` files) - morbo reloads automatically
3. Edit Perl modules (`.pm`) - morbo reloads automatically
4. Edit Tailwind source (`styles/app.css`) - Plugin detects and rebuilds on next request

## Dependencies

### Standalone Tailwind CLI

Download from https://github.com/tailwindlabs/tailwindcss/releases (v3.4.19):

```bash
# Download Linux x64 binary to bin/
curl -sL https://github.com/tailwindlabs/tailwindcss/releases/download/v3.4.19/tailwindcss-linux-x64 -o bin/tailwindcss
chmod +x bin/tailwindcss
```
```

## Notes

- The plugin suppresses the `caniuse-lite is outdated` warning via `BROWSERSLIST_IGNORE_OLD_DATA` env var
- Build output is logged to the Mojolicious log
- CSS is only rebuilt when the source file is newer than the output (using `stat()` modification times)
- Plugin only runs in development mode (`$app->mode eq 'development'`)