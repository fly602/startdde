// SPDX-FileCopyrightText: 2018 - 2022 UnionTech Software Technology Co., Ltd.
//
// SPDX-License-Identifier: GPL-3.0-or-later

package xsettings

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/linuxdeepin/go-lib/utils"
)

const (
	DPI_FALLBACK = 96
	HIDPI_LIMIT  = DPI_FALLBACK * 2

	ffKeyPixels = `user_pref("layout.css.devPixelsPerPx",`
)

// TODO: update 'antialias, hinting, hintstyle, rgba, cursor-theme, cursor-size'
func (m *XSManager) updateDPI() {
	var scale float64
	if !m.dconfigGetValue(gsKeyScaleFactor, &scale) || scale <= 0 {
		scale = 1
	}

	var infos []xsSetting
	var dpi int
	scaledDPI := int32(float64(DPI_FALLBACK*1024) * scale)
	if m.dconfigGetValue("xft-dpi", &dpi) {
		m.dconfigSetValue("xft-dpi", scaledDPI)
		m.gs.SetInt("xft-dpi", scaledDPI)
		infos = append(infos, xsSetting{
			sType: settingTypeInteger,
			prop:  "Xft/DPI",
			value: scaledDPI,
		})
	}

	// update window scale and cursor size
	var windowScale int32
	if !m.dconfigGetValue(gsKeyWindowScale, &windowScale) || windowScale > 1 {
		scaledDPI = int32(DPI_FALLBACK * 1024)
	}

	var cursorSize int32
	if !m.dconfigGetValue(gsKeyGtkCursorThemeSize, &cursorSize) {
		return
	}
	v, _ := m.GetInteger("Gdk/WindowScalingFactor")
	if v != windowScale {
		infos = append(infos, xsSetting{
			sType: settingTypeInteger,
			prop:  "Gdk/WindowScalingFactor",
			value: windowScale,
		}, xsSetting{
			sType: settingTypeInteger,
			prop:  "Gdk/UnscaledDPI",
			value: scaledDPI,
		}, xsSetting{
			sType: settingTypeInteger,
			prop:  "Gtk/CursorThemeSize",
			value: cursorSize,
		})
	}

	if len(infos) != 0 {
		err := m.setSettings(infos)
		if err != nil {
			logger.Warning("Failed to update dpi:", err)
		}
		m.updateXResources()
	}
}

func (m *XSManager) updateXResources() {
	var scaleFactor float64
	if !m.dconfigGetValue(gsKeyScaleFactor, &scaleFactor) || scaleFactor <= 0 {
		scaleFactor = 1
	}
	xftDpi := int(DPI_FALLBACK * scaleFactor)
	var cursorThemeName string
	m.dconfigGetValue("gtk-cursor-theme-name", &cursorThemeName)
	var cursorThemeSize int32
	m.dconfigGetValue(gsKeyGtkCursorThemeSize, &cursorThemeSize)
	updateXResources(xresourceInfos{
		&xresourceInfo{
			key:   "Xcursor.theme",
			value: cursorThemeName,
		},
		&xresourceInfo{
			key:   "Xcursor.size",
			value: fmt.Sprintf("%d", cursorThemeSize),
		},
		&xresourceInfo{
			key:   "Xft.dpi",
			value: strconv.Itoa(xftDpi),
		},
	})
}

var ffDir = path.Join(os.Getenv("HOME"), ".mozilla/firefox")

func (m *XSManager) updateFirefoxDPI() {
	var scale float64
	m.dconfigGetValue(gsKeyScaleFactor, &scale)
	if scale <= 0 {
		// firefox default value: -1
		scale = -1
	}

	configs, err := getFirefoxConfigs(ffDir)
	if err != nil {
		logger.Debug("Failed to get firefox configs:", err)
		return
	}

	for _, config := range configs {
		err = setFirefoxDPI(scale, config, config)
		if err != nil {
			logger.Warning("Failed to set firefox dpi:", config, err)
		}
	}
}

func getFirefoxConfigs(dir string) ([]string, error) {
	finfos, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var configs []string
	for _, finfo := range finfos {
		config := path.Join(dir, finfo.Name(), "prefs.js")
		if !utils.IsFileExist(config) {
			continue
		}
		configs = append(configs, config)
	}
	return configs, nil
}

func setFirefoxDPI(value float64, src, dest string) error {
	contents, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}
	lines := strings.Split(string(contents), "\n")
	target := fmt.Sprintf("%s \"%.2f\");", ffKeyPixels, value)
	found := false
	for i, line := range lines {
		if line == "" || line[0] == '#' {
			continue
		}
		if !strings.Contains(line, ffKeyPixels) {
			continue
		}

		if line == target {
			return nil
		}

		tmp := strings.Split(ffKeyPixels, ",")[0] + ", " +
			fmt.Sprintf("\"%.2f\");", value)
		lines[i] = tmp
		found = true
		break
	}
	if !found {
		if value == -1 {
			return nil
		}
		tmp := lines[len(lines)-1]
		lines[len(lines)-1] = target
		lines = append(lines, tmp)
	}
	return ioutil.WriteFile(dest, []byte(strings.Join(lines, "\n")), 0644)
}
